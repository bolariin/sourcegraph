# Postgres Migrations

The children of this directory contain migrations for each Postgres database instance:

- `frontend` is the main database (things should go here unless there is a good reason)
- `codeintel` is a database containing only processed LSIF data (which can become extremely large)
- `codeinsights` is a TimescaleDB database, containing only Code Insights time series data.

The migration path for each database instance is the same and is described below. Each of the database instances described here are deployed separately, but are designed to be _overlayable_ to reduce friction during development. That is, we assume that the names in each database do not overlap so that the same connection parameters can be used for both database instances. Each database also has a uniquely named schema versions table:

| database       | schema version table name        |
| -------------- | -------------------------------- |
| `frontend`     | `schema_migrations`              |
| `codeintel`    | `codeintel_schema_migrations`    |
| `codeinsights` | `codeinsights_schema_migrations` |

Migrations are handled by the [migrate](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate#installation) tool. Migrations get applied automatically at application startup. The CLI tool can also be used to manually test migrations.

## Migrating up and down

Up migrations happen automatically on server start-up. They can also be run manually using the migrate CLI:

- run `./dev/db/migrate.sh <db_name> -h` for a list of options
- run `./dev/db/migrate.sh <db_name> up` to move forward to the latest migration
- run `./dev/db/migrate.sh <db_name> down 1` to rollback the previous migration

If a migration fails, and you need to revert to a previous state `./dev/db/migrate.sh <db_name> force` may be helpful. Alternatively use the `dropdb` and `createdb` commands to wipe your local database and start from a clean state.

**Note:** if you find that you need to run a down migration, that almost certainly means the migration was not backward-compatible, and you should fix this before merging the migration into `main`.

## Adding a migration

**IMPORTANT:** All migrations must be backward-compatible, meaning that an _existing_ running instance must be able to run against the _new_ (post-migration) version of the schema. This is because frontend pods are updated in a rolling fashion. During the rolling update, there will be both old and new frontend pods. The first updated pod will migrate the schema atomically, but the remaining old ones may continue to write before they are terminated.

Run the following:

```
./dev/db/add_migration.sh <db_name> my_migration_name
```

There will be up/down `.sql` migration files created in the instance's migrations directory. Add SQL statements to these files that will perform the desired migration.

**NOTE**: the migration runner does not use transactions. Use the explicit transaction blocks added to the migration script template.

After adding SQL statements to those files, update the schema doc:

```
go generate ./internal/database/
```

Alternatively, regenerate everything in the repository via `./dev/generate.sh`.

Verify that the migration is backward-compatible. We currently have no automated testing for this. You need to ensure that an old version of Sourcegraph, like what is currently deployed on Sourcegraph.com, can continue to use the DB during a rolling upgrade from the old version to your version.

Some migrations are difficult to do in a single step. For instance, renaming a column, table, or view, or adding a column with a non-nullable constraint will all break existing code that accesses that table or view. In order to do such changes you may need to break your changes into several parts separated by a deployment.

For example, a non-nullable column can be added to an existing table with the following steps:

- Add a nullable column to the table
- Update the code to always populate this row on writes
- Deploy to Sourcegraph.com
- Add a non-nullable constraint to the table
- Deploy to Sourcegraph.com

We have a hard requirement (enforced by CI) that rolling upgrades are always possible on Sourcegraph.com. When possible, this same standard should be kept between minor release versions to ensure a smooth upgrade process for private instances (although there will be exceptions due to feature velocity and a monthly release cadence).

### Rebasing a migration

On longer running branches, you might find that your migration now conflicts with another migration added while you were working on your branch. Don't despair! Here are some handy tips when rebasing a branch on `main` that has a migration conflict:

1. It's usually easiest to separate out your migration into a separate commit, with nothing else in it. (You probably want this to be the first commit on your branch, for rebasing simplicity.)
2. Before you rebase, you should migrate down to the version before your migration. `./dev/db/migrate.sh <database> down 1` will usually take care of this for you.
3. Once you start rebasing, you'll get an error like this on your migration commit:

   ```
   Auto-merging internal/database/schema.md
   error: could not apply 4931031d10... Add migrations.
   Resolve all conflicts manually, mark them as resolved with
   "git add/rm <conflicted_files>", then run "git rebase --continue".
   You can instead skip this commit: run "git rebase --skip".
   To abort and get back to the state before "git rebase", run "git rebase --abort".
   Could not apply 4931031d10... Add migrations.
   ```

   We need to renumber your migration.

4. You can renumber your migration by `git mv`-ing the relevant up and down files, or with this script: `./dev/db/rebase_migration.sh <database> <either your up or down file>`
5. Once done, you need to regenerate the schema. If you use `rebase_migration.sh`, it will suggest what to do, but it's roughly:

   ```bash
   ./dev/db/migrate.sh <database> up
   go generate ./internal/database
   ```

6. From there, `git add` your updated files, and you should be able to continue your rebase.

## Customer rollbacks

Running down migrations in a rollback **should NOT** be necessary if all migrations are backward-compatible. In case the customer must run a down migration, they will need perform do the following steps.

- Roll back Sourcegraph to the previous version. On startup, the frontend pods will log a migration warning stating that the schema has been migrated to a newer version. This warning should **NOT** indicate that the database is dirty.

- Determine if a database is dirty by running the following commands.

  **frontend database**:

  ```
  kubectl exec $(kubectl get pod -l app=pgsql -o jsonpath='{.items[0].metadata.name}') -- psql -U sg -c 'SELECT * FROM schema_migrations'
  ```

  **codeintel database**:

  ```
  kubectl exec $(kubectl get pod -l app=pgsql-codeintel -o jsonpath='{.items[0].metadata.name}') -- psql -U sg -c 'SELECT * FROM codeintel_schema_migrations'
  ```

  **codeinsights database**:

  ```
  kubectl exec $(kubectl get pod -l app=codeinsights-db -o jsonpath='{.items[0].metadata.name}') -- psql -U sg -c 'SELECT * FROM codeinsights_schema_migrations'
  ```

  For each dirty database, follow the steps in the _Dirty schema_ section below.

- For each database `<db_name>` with the schema version table `<schema_version_table_name>`, do the following:
  - Determine the two commits that correspond to the previous and new versions of Sourcegraph. Check out each commit and run `ls -1` in the `migrations/<db_name>` directory. The order of the migrations is the same as the alphabetical order of the migration scripts, so take the diff between the two list outputs to determine which migrations should be run.
  - Apply the down migration scripts in **reverse chronological order**. Wrap each down migration in a transaction block. If there are any errors, stop and resolve the issue before proceeding with the next down migration.
  - After all down migrations have been applied, run
    ```
    update <schema_version_table_name> set version=$VERSION;
    ```
    where `$VERSION` is the numerical prefix of the migration script corresponding to the first migration you _didn't_ just apply. In other words, it is the numerical prefix of the last migration script as of the rolled-back-to commit.
  - Restart frontend pods. On restart, they should spin up successfully.

### Reverting a migration

If a PR which contains a DB migration was reverted, it may still have been applied to Sourcegraph.com, k8s.sgdev.org, etc. due to their rollout schedules. In some cases, it may also have been part of a Sourcegraph release.

To fix this, you should create a PR to revert the migration from the DB. Say the migration files were:

- `1234_do_something.up.sql`
- `1234_do_something.down.sql`

You should then:

1. Rename the files to `1234_reverted.up.sql` and `1234_reverted.down.sql`
2. Replace the contents of those files with just:

```sql
BEGIN;

-- This migration was reverted, see: <github issue link>

COMMIT;
```

3. Add a new migration using `./dev/db/add_migration.sh <database> undo_something` which will consume the next sequentialm migration ID.
4. Your new `.up.sql` migration should contain the contents of the old `1234_reverted.down.sql` and you will need to update the migration to run down migrations idempotently, i.e. using `IF EXISTS` etc. everywhere as _some instances running it may not have run the up migration_.
5. Your new `.down.sql` should be an empty migration.

For an example of how this looks: https://github.com/sourcegraph/sourcegraph/pull/25717

## Troubleshooting

### Dirty schema

If the schema for a database is dirty, that means the current migration (as indicated in the schema version table), regardless of migration direction) failed midway through. This should almost never happen. If it does happen, it probably means up/down migrations were applied out of order or other manual changes were made to the DB that conflict with the current migration stage.

If the schema for database `<db_name>` with the schema version table `<schema_version_table_name>` is dirty, do the following:

- Figure out what change was made to cause the migration to fail midway through.
- Let `$VERSION` be the numerical prefix of the last migration script run to produce this version of the schema.
- Run the necessary SQL commands to make the schema consistent with version `$VERSION` of the schema.
- Run `update <schema_version_table_name> set version=$VERSION, dirty=false;`.
- Restart frontend pods.
