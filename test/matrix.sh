#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
compose_file="$repo_root/test/compose.yaml"
bridge_image=${FORGEJO_BRIDGE_IMAGE:-forgejo-bridge:mvp}
export COMPOSE_PROJECT_NAME=forgejo-bridge-matrix
network_name=${COMPOSE_PROJECT_NAME}_default
matrix_tmp=$(mktemp -d)

cleanup() {
	docker compose -f "$compose_file" down -v >/dev/null 2>&1 || true
	rm -rf -- "$matrix_tmp"
}
trap cleanup EXIT INT TERM

docker compose -f "$compose_file" up -d --wait forgejo15 forgejo16

for service in forgejo15 forgejo16; do
	docker compose -f "$compose_file" exec -T "$service" forgejo admin user create \
		--username bridge \
		--password 'Bridge-Test-Only-42!' \
		--email bridge@example.invalid \
		--must-change-password=false >/dev/null
	token=$(docker compose -f "$compose_file" exec -T "$service" forgejo admin user generate-access-token \
		--username bridge --token-name bridge-matrix --raw --scopes all)

	status=$(docker run --rm --network "$network_name" curlimages/curl:8.12.1 \
		-sS -o /dev/null -w '%{http_code}' -X POST "http://$service:3000/api/v1/user/repos" \
		-H "Authorization: token $token" -H 'Content-Type: application/json' \
		--data '{"name":"contract","auto_init":true,"default_branch":"main"}')
	test "$status" = 201
	status=$(docker run --rm --network "$network_name" curlimages/curl:8.12.1 \
		-sS -o /dev/null -w '%{http_code}' -X POST "http://$service:3000/api/v1/repos/bridge/contract/issues" \
		-H "Authorization: token $token" -H 'Content-Type: application/json' \
		--data '{"title":"matrix issue","body":"black-box contract"}')
	test "$status" = 201

	config="$matrix_tmp/$service.toml"
	printf 'default_instance = "matrix"\n[instances.matrix]\nbase_url = "http://%s:3000"\ncredential = "env:FORGEJO_TOKEN"\nallowed_repositories = ["bridge/contract"]\nread_only = true\n' "$service" > "$config"

	docker run --rm --network "$network_name" -e "FORGEJO_TOKEN=$token" -v "$config:/config.toml:ro" \
		"$bridge_image" branch list --config /config.toml bridge/contract >/dev/null
	docker run --rm --network "$network_name" -e "FORGEJO_TOKEN=$token" -v "$config:/config.toml:ro" \
		"$bridge_image" issue list --state all --config /config.toml bridge/contract >/dev/null

	if [ "$service" = forgejo16 ]; then
		docker run --rm --network "$network_name" -e "FORGEJO_TOKEN=$token" -v "$config:/config.toml:ro" \
			"$bridge_image" actions run list --config /config.toml bridge/contract >/dev/null
	else
		set +e
		docker run --rm --network "$network_name" -e "FORGEJO_TOKEN=$token" -v "$config:/config.toml:ro" \
			"$bridge_image" actions run list --config /config.toml bridge/contract >/dev/null
		status=$?
		set -e
		test "$status" = 9
	fi
done

echo "Forgejo 15/16 bridge matrix passed"
