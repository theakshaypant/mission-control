.PHONY: test test-e2e test-e2e-github lint

# Run all unit tests.
test:
	go test ./...

# Run all e2e test suites (requires env vars per suite).
test-e2e:
	go test -v -tags integration ./test/e2e/...

# Run only the GitHub e2e tests.
# Requires: GITHUB_TOKEN, GITHUB_USER, GITHUB_TEST_REPOS
test-e2e-github:
	go test -v -tags integration ./test/e2e/github/

# Run a specific GitHub e2e test by name.
# Usage: make test-e2e-github-run TEST=TestGitHubPRSignalUnreviewed
test-e2e-github-run:
	go test -v -tags integration -run $(TEST) ./test/e2e/github/

lint:
	go vet ./...
