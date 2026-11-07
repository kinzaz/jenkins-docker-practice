#!/usr/bin/env bash
set -euo pipefail

cd /home/devops/docker-jenkins

tag="${1:-}"
simulate_failure="${2:-false}"

if [[ ! "$tag" =~ ^jenkins-[0-9]+$ ]]; then
    echo "Expected tag like jenkins-16" >&2
    exit 1
fi

if [[ "$simulate_failure" != true && "$simulate_failure" != false ]]; then
    echo "Second argument must be true or false" >&2
    exit 1
fi

exec 9>.deploy.lock
flock -n 9 || {
    echo "Another deployment is running" >&2
    exit 1
}

compose_api() {
    sudo -n /usr/bin/docker compose \
        --project-directory /home/devops/docker-jenkins \
        -f /home/devops/docker-jenkins/compose.yaml \
        up -d --no-deps --pull never api
}

check_release() {
    local expected="$1"
    local release
    local expected_json

    curl --fail --silent --show-error \
        --retry 10 --retry-delay 2 --retry-all-errors \
        --connect-timeout 2 --max-time 5 \
        http://127.0.0.1:8080/api/health || return 1

    release=$(
        curl --fail --silent --show-error \
            --retry 10 --retry-delay 2 --retry-all-errors \
            --connect-timeout 2 --max-time 5 \
            http://127.0.0.1:8080/api/version
    ) || return 1

    expected_json=$(printf '{"version":"%s"}' "$expected")
    if [[ "$release" != "$expected_json" ]]; then
        echo "Release mismatch: expected $expected, received $release" >&2
        return 1
    fi

    echo
    echo "Verified backend release: $release"
}

previous_tag=$(awk -F= '$1 == "API_TAG" {print $2}' .env)

if [[ ! "$previous_tag" =~ ^jenkins-[0-9]+$ ]]; then
    echo "Expected an existing Jenkins release in .env" >&2
    exit 1
fi

# Проверяем исходный выпуск до изменения конфигурации.
check_release "$previous_tag"

backup=$(mktemp .env.backup.XXXXXX)
env_temp=$(mktemp .env.next.XXXXXX)
cp -p .env "$backup"
rollback_needed=false

finish() {
    local status=$?
    trap - EXIT

    if [[ "$rollback_needed" == true ]]; then
        echo "Deployment failed. Restoring $previous_tag" >&2

        if mv "$backup" .env &&
            compose_api &&
            check_release "$previous_tag"; then
            echo "Rollback completed: $previous_tag"
        else
            echo "ROLLBACK FAILED. Manual recovery required." >&2
        fi

        status=1
    fi

    rm -f "$backup" "$env_temp"
    exit "$status"
}

trap finish EXIT

awk -v tag="$tag" '
    !/^[[:space:]]*(export[[:space:]]+)?API_TAG[[:space:]]*=/ {
        print
    }
    END {
        print "API_TAG=" tag
    }
' .env > "$env_temp"

chmod 600 "$env_temp"

rollback_needed=true
mv "$env_temp" .env

compose_api
check_release "$tag"

if [[ "$simulate_failure" == true ]]; then
    echo "Simulated deployment failure" >&2
    exit 1
fi

rollback_needed=false
echo "Backend deployment check passed for $tag"
