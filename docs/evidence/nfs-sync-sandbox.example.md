# NFS Sync Sandbox Evidence

- Verdict: UNVERIFIED
- Date/time:
- Host/kernel:
- Commit or diff reference:
- Operator:

## Target environment
- Source NFS host/export:
- Destination NFS host/export:
- Source test subpath:
- Destination test subpath:
- Source verify root (`FVS_TEST_SRC_VERIFY_ROOT`):
- Destination verify root (`FVS_TEST_DST_VERIFY_ROOT`):
- Privilege mode (root/CAP_SYS_ADMIN/etc.):
- CSV `source_volume_key`: dummy non-secret placeholder only
- Verify roots: private, disposable, owned by privileged test user, not group/world writable, no symlinked path components

## Command
```bash
sudo env -i \
  PATH='/usr/sbin:/usr/bin:/sbin:/bin' \
  SRC_STORAGE_MOUNT_HOST='<source-nfs-host>' \
  SRC_STORAGE_MOUNT_OPTION='<source-mount-options>' \
  SRC_STORAGE_MOUNT_NAME='src' \
  DST_STORAGE_MOUNT_HOST='<destination-nfs-host>' \
  DST_STORAGE_MOUNT_OPTION='<destination-mount-options>' \
  DST_STORAGE_MOUNT_NAME='dst' \
  REPORT_ENABLED=true \
  SCAN_FIND_PATH='' \
  RETRY_ATTEMPTS=0 \
  /path/to/fast-volume-syncer select 0 /path/to/one-row.csv
```

## Log
- Log path: `/path/to/log`
- Redactions applied (`source_volume_key`, credentials, tokens, unrelated secrets):
- Attached stdout/stderr excerpt or reference:

## Mount evidence
```text
<findmnt output for source mount>
<findmnt output for destination mount>
```

## Data verification
```bash
sha256sum <source-mounted-file>
sha256sum <destination-mounted-file>
readlink <source-mounted-link>
readlink <destination-mounted-link>
```

## Cleanup evidence
```text
<findmnt | grep fast-volume-syncer || true>
<findmnt | grep syncer- || true>
```

## Notes
- The disposable selector CSV must use a dummy non-secret `source_volume_key` value.
- Redact `source_volume_key`, credentials, tokens, and unrelated secrets before sharing logs or pasted command snippets.
- Missing evidence or failures:
