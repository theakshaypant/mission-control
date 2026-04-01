EXT_UUID   = mission-control@theakshaypant
EXT_DIR    = $(HOME)/.local/share/gnome-shell/extensions/$(EXT_UUID)
IMAGE_NAME ?= mission-control

.PHONY: test test-e2e test-e2e-github lint install-gnome-ext docker-build

# Run all unit tests.
test:
	go test -count=1 ./...

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

# Build the single-container image (Go API + React dashboard).
# Usage: make docker-build [IMAGE_NAME=my-tag]
docker-build:
	docker build -t $(IMAGE_NAME) .

# Run the container, mounting the host config directory and matching the host
# user so file permissions on the mount work without any chown dance.
docker-run:
	docker run -d --rm -p 5040:5040 \
		--name $(IMAGE_NAME) \
		--user $$(id -u):$$(id -g) \
		-v $(HOME)/.config/mission-control:/config \
		$(IMAGE_NAME) -config /config/config.yaml

# First install: copy files, then log out and back in (required on Wayland),
# then run: gnome-extensions enable $(EXT_UUID)
#
# Subsequent updates (extension already enabled):
#   make update-gnome-ext
install-gnome-ext:
	mkdir -p $(EXT_DIR)
	cp -r gnome-extension/. $(EXT_DIR)/
	glib-compile-schemas $(EXT_DIR)/schemas/
	@echo "Files copied. Log out and back in, then run:"
	@echo "  gnome-extensions enable $(EXT_UUID)"

# Update an already-enabled extension without logging out.
update-gnome-ext:
	cp -r gnome-extension/. $(EXT_DIR)/
	glib-compile-schemas $(EXT_DIR)/schemas/
	gnome-extensions disable $(EXT_UUID)
	gnome-extensions enable $(EXT_UUID)
