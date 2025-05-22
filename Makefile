# NOTE: Change necessary details
DB_STRING=user=gihyun dbname=resqlink password=password host=0.0.0.0 sslmode=disable

db-up:
	@GOOSE_DRIVER=postgres GOOSE_DBSTRING="$(DB_STRING)" GOOSE_MIGRATION_DIR="./database/migrations/" goose up

db-down:
	@GOOSE_DRIVER=postgres GOOSE_DBSTRING="$(DB_STRING)" GOOSE_MIGRATION_DIR="./database/migrations/" goose down

db-create:
ifeq ($(name),)
	$(error `name` is not set. Usage: `make db-create name="migration name"`)
endif
	@GOOSE_DRIVER=postgres GOOSE_DBSTRING="$(DB_STRING)" GOOSE_MIGRATION_DIR="./database/migrations/" goose create "$(name)" sql


