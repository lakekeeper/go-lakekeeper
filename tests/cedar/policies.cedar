// Server policies for the Lakekeeper+ compose stack (profile plus, Cedar
// authorization backend). Bootstrap grants nothing under Cedar, so the admin
// rights come from here.

// The lakekeeper-admin client's service account from
// tests/keycloak-realm.json may do everything. Change both together.
permit (
    principal == Lakekeeper::User::"oidc~6deeb417-cdf9-4320-8a30-ddecea77a4bd",
    action,
    resource
);
