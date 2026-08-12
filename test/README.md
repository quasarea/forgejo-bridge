# Forgejo integration matrix

`compose.yaml` pins the currently certified Forgejo 15 LTS and Forgejo 16
versions. Unit HTTP fixtures exercise the normalized resource contract and
version gates without depending on mutable external data. Version startup can
be checked with:

```sh
docker compose -f test/compose.yaml up -d --wait
curl -fsS http://localhost:13015/api/v1/version
curl -fsS http://localhost:13016/api/v1/version
docker compose -f test/compose.yaml down -v
```

Run this only in an isolated test environment. The matrix creates and removes
containers, volumes, users, repositories, and ephemeral access tokens.

For the authenticated black-box contract check, first build the bridge image
and then run:

```sh
docker build -t forgejo-bridge:mvp .
sh test/matrix.sh
```

The script creates disposable users, test data, ephemeral tokens, containers, and
volumes, verifies branches/issues on both versions plus the Actions version
gate, and removes the complete test environment on exit.
