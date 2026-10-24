# Audit

The audit service logs all events of the system as an audit log. Per default, it will be logged to standard out, but can also be configured to a file output. Supported log formats are json or a minimal human-readable format.

With audit logs, you are able to prove compliance with corporate guidelines as well as to enable reporting and auditing of operations. The audit service takes note of actions conducted by users and administrators.

Example minimal format:
```
file_delete)
   user 'user_id' trashed file 'item_id'
file_trash_delete)
   user 'user_id' removed file 'item_id' from trashbin
file_read)
   user 'user_id' read file 'item_id'
```

Example json:
```
{"RemoteAddr":"","User":"user_id","URL":"","Method":"","UserAgent":"","Time":"","App":"admin_audit","Message":"user 'user_id' trashed file 'item_id'","Action":"file_delete","CLI":false,"Level":1,"Path":"path","Owner":"user_id","FileID":"item_id"}
{"RemoteAddr":"","User":"user_id","URL":"","Method":"","UserAgent":"","Time":"","App":"admin_audit","Message":"user 'user_id' removed file 'item_id' from trashbin","Action":"file_trash_delete","CLI":false,"Level":1,"Path":"path","Owner":"user_id","FileID":"item_id"}
{"RemoteAddr":"","User":"user_id","URL":"","Method":"","UserAgent":"","Time":"","App":"admin_audit","Message":"user 'user_id' read file 'item_id'","Action":"file_read","CLI":false,"Level":1,"Path":"path","Owner":"user_id","FileID":"item_id"}
```

The audit service is not started automatically when running as single binary started via `opencloud server` or when running as docker container and must be started and stopped manually on demand.

The audit service logs:

- File system operations
(create/delete/move/read; including actions on the trash bin and versioning)
- User management operations
(creation/deletion of users)
- Sharing operations
(user/group sharing, sharing via link, changing permissions, calls to sharing API from clients)
