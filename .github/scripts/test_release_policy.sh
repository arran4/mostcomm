#!/bin/bash
set -euo pipefail

echo "Running release policy non-mutating regression coverage tests"

FAIL=0

test_validate_tag() {
    local tag=$1
    local expected_fail=$2

    echo "Testing tag format: $tag (Expected to fail: $expected_fail)"

    # Policy check from publisher step
    if [[ "$tag" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
        if [ "$expected_fail" = true ]; then
            echo "FAIL: Tag $tag should have been rejected by regex."
            FAIL=1
        else
            echo "PASS"
        fi
    else
        if [ "$expected_fail" = true ]; then
            echo "PASS"
        else
            echo "FAIL: Tag $tag should have been accepted by regex."
            FAIL=1
        fi
    fi
}

test_validate_tag "v1.0.0" false
test_validate_tag "v1.2.3" false
test_validate_tag "v0.1.2" false
test_validate_tag "v01.2.3" true
test_validate_tag "v1.02.3" true
test_validate_tag "v1.2.3-rc1" true
test_validate_tag "1.2.3" true
test_validate_tag "test" true

if [ $FAIL -eq 0 ]; then
    echo "All release policy tests passed."
    exit 0
else
    echo "Release policy tests failed."
    exit 1
fi
