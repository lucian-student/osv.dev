# Gitter

Honestly from what i see, this service isnt doing much, so there shouldnt need to be any reason to need that much resources.

## GetGit streams down a compressed tarball archive of the target repository.

Unused

## Cache triggers proactive caching of a target repository.

Unused

## GetTags returns the full list of tags/refs mapped to their commit hashes.

Enriches some Github_Actions ecosystem apparently, so far i dont know how.

## GetAffectedCommits resolves and lists affected commits based on introduced/fixed/last_affected commit ranges.

Unused

## GetFileDiffs retrieves the file changes since given last synced commit.

It is called in the importer, when loading new changes from datasources:
1. this happens in the cron job, that happens only once 15mins, so thats probably not necessery to optimize for

## GetFileContent retrieves the raw, uncompressed content of a single file path.

This is used in recoverer.(NOt sure if performance of recoverer is neccessery, because i dont even know what recoverer does.)

Importer, from my guess when updating documents, so maybe reconciler process?

## GetCommitDiffs retrieves structured commit data and diff patches since a given last synced commit or timestamp.

Unused