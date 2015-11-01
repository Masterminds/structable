# schema2struct: Create definitions from the database

This program is a proof of concept for creating Structable structs by
inspecting a database and generating closely matching structs.

Currently this only works on Postgres, though there is no reason it
could not be ported to support other databases.

It works by querying the INFORMATION_SCHEMA tables to learn about what
tables are present and what columns they stored. It then attempts to
render structs that point to those tables.

If you are interested in contributing to moving this beyond proof of
concept, feel free to issue PRs against the codebase.
