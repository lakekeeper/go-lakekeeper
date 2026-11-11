#!/usr/bin/env node
// Export the GitHub images of a finished Archify render.
//
// Usage: node scripts/export-diagram-svg.mjs <render-folder>/<slug>.html
//
// The command opens a finished Archify render's HTML in headless Chrome, runs
// the viewer's own "SVG (light)" and "SVG (dark)" exports, and writes
// <slug>-light.svg and <slug>-dark.svg next to it. Each SVG carries its styles
// and font inline, so it renders on its own (e.g. on GitHub).
// scripts/render-diagrams.sh calls it for each render.
//
// Needs Node 18+ and Chrome or Chromium (set ARCHIFY_CHROME to override the
// lookup, as for Archify itself). It does not need Archify.

import { spawn } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { pathToFileURL } from 'node:url';

const THEMES = ['light', 'dark'];
const TIMEOUT_MS = 60000;

function fail(message) {
  process.stderr.write(`export-diagram-svg: ${message}\n`);
  process.exit(1);
}

function findChrome() {
  if (process.env.ARCHIFY_CHROME) return process.env.ARCHIFY_CHROME;
  const fixed = process.platform === 'darwin'
    ? ['/Applications/Google Chrome.app/Contents/MacOS/Google Chrome', '/Applications/Chromium.app/Contents/MacOS/Chromium']
    : [];
  const commands = ['google-chrome', 'google-chrome-stable', 'chromium', 'chromium-browser', 'chrome'];
  for (const dir of (process.env.PATH || '').split(path.delimiter)) {
    for (const command of commands) fixed.push(path.join(dir, command));
  }
  return fixed.find((candidate) => {
    try {
      fs.accessSync(candidate, fs.constants.X_OK);
      return fs.statSync(candidate).isFile();
    } catch {
      return false;
    }
  });
}

// Chrome DevTools Protocol over --remote-debugging-pipe: NUL-terminated JSON on fds 3 and 4.
class Cdp {
  constructor(child) {
    this.nextId = 1;
    this.pending = new Map();
    this.listeners = [];
    this.buffer = '';
    this.out = child.stdio[3];
    child.stdio[4].setEncoding('utf8');
    child.stdio[4].on('data', (chunk) => this.consume(chunk));
    child.once('exit', (code) => this.failAll(new Error(`Chrome exited with code ${code}`)));
  }

  consume(chunk) {
    this.buffer += chunk;
    let end;
    while ((end = this.buffer.indexOf('\0')) >= 0) {
      const message = JSON.parse(this.buffer.slice(0, end));
      this.buffer = this.buffer.slice(end + 1);
      const pending = message.id && this.pending.get(message.id);
      if (pending) {
        this.pending.delete(message.id);
        if (message.error) pending.reject(new Error(`${pending.method}: ${message.error.message}`));
        else pending.resolve(message.result || {});
      } else if (message.method) {
        for (const listener of [...this.listeners]) listener(message);
      }
    }
  }

  send(method, params = {}, sessionId) {
    const id = this.nextId++;
    return new Promise((resolve, reject) => {
      this.pending.set(id, { method, resolve, reject });
      this.out.write(`${JSON.stringify({ id, method, params, sessionId })}\0`);
    });
  }

  once(method, sessionId) {
    return new Promise((resolve) => {
      const listener = (message) => {
        if (message.method !== method || message.sessionId !== sessionId) return;
        this.listeners.splice(this.listeners.indexOf(listener), 1);
        resolve(message.params);
      };
      this.listeners.push(listener);
    });
  }

  failAll(error) {
    for (const pending of this.pending.values()) pending.reject(error);
    this.pending.clear();
  }
}

// Runs in the viewer page: waits until the diagram is laid out, then clicks the
// export menu's SVG item for the theme and returns the SVG the viewer produced.
function exportInPage(theme) {
  const stable = () => {
    const fonts = document.fonts ? document.fonts.ready.catch(() => {}) : Promise.resolve();
    const layout = window.Archify && Archify.layoutStability && Archify.layoutStability.whenStable;
    return fonts.then(() => (layout ? Archify.layoutStability.whenStable() : undefined));
  };
  return stable().then(() => new Promise((resolve, reject) => {
    const button = document.querySelector(`button[data-format="svg-${theme}"]`);
    if (!button) {
      reject(new Error(`the viewer has no "svg-${theme}" export; is this an Archify render?`));
      return;
    }
    const createObjectURL = URL.createObjectURL;
    URL.createObjectURL = function (blob) {
      URL.createObjectURL = createObjectURL;
      blob.text().then(resolve, reject);
      return createObjectURL.call(URL, blob);
    };
    button.click();
    setTimeout(() => {
      const error = document.documentElement.getAttribute('data-last-export-error');
      reject(new Error(error || 'the viewer produced no SVG within 30s'));
    }, 30000);
  }));
}

async function exportSvgs(chromePath, html) {
  const profile = fs.mkdtempSync(path.join(os.tmpdir(), 'export-diagram-svg-'));
  const args = [
    '--headless=new', '--remote-debugging-pipe', '--disable-gpu', '--no-first-run',
    '--no-default-browser-check', '--disable-background-networking', '--disable-sync',
    `--user-data-dir=${profile}`, 'about:blank',
  ];
  if (process.env.ARCHIFY_CHROME_NO_SANDBOX === '1') args.unshift('--no-sandbox');
  const child = spawn(chromePath, args, { stdio: ['ignore', 'ignore', 'ignore', 'pipe', 'pipe'] });
  const cdp = new Cdp(child);
  const timer = setTimeout(() => {
    child.kill();
    fail(`timed out after ${TIMEOUT_MS / 1000}s exporting ${html}`);
  }, TIMEOUT_MS);
  try {
    const { targetId } = await cdp.send('Target.createTarget', { url: 'about:blank' });
    const { sessionId } = await cdp.send('Target.attachToTarget', { targetId, flatten: true });
    await cdp.send('Page.enable', {}, sessionId);
    await cdp.send('Emulation.setDeviceMetricsOverride', { width: 1600, height: 1000, deviceScaleFactor: 1, mobile: false }, sessionId);
    const loaded = cdp.once('Page.loadEventFired', sessionId);
    const navigation = await cdp.send('Page.navigate', { url: pathToFileURL(path.resolve(html)).href }, sessionId);
    if (navigation.errorText) throw new Error(`cannot open ${html}: ${navigation.errorText}`);
    await loaded;
    const svgs = {};
    for (const theme of THEMES) {
      const response = await cdp.send('Runtime.evaluate', {
        expression: `(${exportInPage})(${JSON.stringify(theme)})`,
        awaitPromise: true,
        returnByValue: true,
      }, sessionId);
      if (response.exceptionDetails) {
        throw new Error(response.exceptionDetails.exception?.description || response.exceptionDetails.text);
      }
      svgs[theme] = response.result.value;
    }
    return svgs;
  } finally {
    clearTimeout(timer);
    // Chrome keeps writing its profile while it shuts down: wait for it to exit.
    const exited = child.exitCode === null ? new Promise((resolve) => child.once('exit', resolve)) : null;
    child.kill();
    await exited;
    fs.rmSync(profile, { recursive: true, force: true, maxRetries: 5 });
  }
}

const html = process.argv[2];
if (!html || process.argv.length > 3 || !html.endsWith('.html')) fail('usage: node scripts/export-diagram-svg.mjs <render-folder>/<slug>.html');
if (!fs.existsSync(html) || !fs.statSync(html).isFile()) fail(`${html} not found; render the diagram with Archify first`);
const folder = path.dirname(html);
const slug = path.basename(html, '.html');
const chromePath = findChrome();
if (!chromePath) fail('Chrome or Chromium not found; install it or set ARCHIFY_CHROME to its executable');

try {
  const svgs = await exportSvgs(chromePath, html);
  for (const theme of THEMES) {
    if (!/^<\?xml[^>]*\?>\s*<svg[\s>]/.test(svgs[theme] || '')) fail(`the ${theme} export is not an SVG document`);
    const target = path.join(folder, `${slug}-${theme}.svg`);
    fs.writeFileSync(target, svgs[theme]);
    process.stdout.write(`${target}\n`);
  }
} catch (error) {
  fail(error.message);
}
