.PHONY: bump
# this only bumps feat and fix
bump:
	@echo "Bumping version..."
	@cog bump -a
# this uses the major profile to bump the version regardless of the current version. Useful when you want to force a major version bump e.g 'refactor!: rename methods'
bump-major:
	@echo "Bumping major version..."
	@cog bump -M

test:
	@echo "Running tests..."
	@go test ./...