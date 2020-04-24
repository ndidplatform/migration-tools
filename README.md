# NDID Migration Tools

## Migrate Data to a New Chain

1. Run backup in `backup_<SOURCE_VERSION>_for_<DESTINATION_VERSION>` sub repo

2. Run restore in `restore_to_<DESTINATION_VERSION>` sub repo

## TODOs

- Refactor and separate logic for each version
- Data conversion from one version to another version module (allow version skipping e.g. v3 to v5)
- Combine backup/restore of specific version to one and able to specify source version and destination version
