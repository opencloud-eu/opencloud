## Scenarios from core API tests that are expected to fail with decomposed storage while running with the Graph API

### File

Basic file management like up and download, move, copy, properties, trash, versions and chunking.

#### [Custom dav properties with namespaces are rendered incorrectly](https://github.com/owncloud/ocis/issues/2140)

_ocdav: double-check the webdav property parsing when custom namespaces are used_

- [coreApiWebdavProperties/setFileProperties.feature:128](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavProperties/setFileProperties.feature#L128)
- [coreApiWebdavProperties/setFileProperties.feature:129](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavProperties/setFileProperties.feature#L129)
- [coreApiWebdavProperties/setFileProperties.feature:130](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavProperties/setFileProperties.feature#L130)

### Sync

Synchronization features like etag propagation, setting mtime and locking files

#### [Uploading an old method chunked file with checksum should fail using new DAV path](https://github.com/owncloud/ocis/issues/2323)

- [coreApiMain/checksums.feature:233](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiMain/checksums.feature#L233)
- [coreApiMain/checksums.feature:234](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiMain/checksums.feature#L234)
- [coreApiMain/checksums.feature:235](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiMain/checksums.feature#L235)

### Share

#### [d:quota-available-bytes in dprop of PROPFIND give wrong response value](https://github.com/owncloud/ocis/issues/8197)

- [coreApiWebdavProperties/getQuota.feature:57](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavProperties/getQuota.feature#L57)
- [coreApiWebdavProperties/getQuota.feature:58](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavProperties/getQuota.feature#L58)
- [coreApiWebdavProperties/getQuota.feature:59](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavProperties/getQuota.feature#L59)
- [coreApiWebdavProperties/getQuota.feature:73](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavProperties/getQuota.feature#L73)
- [coreApiWebdavProperties/getQuota.feature:74](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavProperties/getQuota.feature#L74)
- [coreApiWebdavProperties/getQuota.feature:75](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavProperties/getQuota.feature#L75)

#### [deleting a file inside a received shared folder is moved to the trash-bin of the sharer not the receiver](https://github.com/owncloud/ocis/issues/1124)

- [coreApiTrashbin/trashbinSharingToShares.feature:54](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiTrashbin/trashbinSharingToShares.feature#L54)
- [coreApiTrashbin/trashbinSharingToShares.feature:55](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiTrashbin/trashbinSharingToShares.feature#L55)
- [coreApiTrashbin/trashbinSharingToShares.feature:56](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiTrashbin/trashbinSharingToShares.feature#L56)
- [coreApiTrashbin/trashbinSharingToShares.feature:83](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiTrashbin/trashbinSharingToShares.feature#L83)
- [coreApiTrashbin/trashbinSharingToShares.feature:84](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiTrashbin/trashbinSharingToShares.feature#L84)
- [coreApiTrashbin/trashbinSharingToShares.feature:85](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiTrashbin/trashbinSharingToShares.feature#L85)
- [coreApiTrashbin/trashbinSharingToShares.feature:142](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiTrashbin/trashbinSharingToShares.feature#L142)
- [coreApiTrashbin/trashbinSharingToShares.feature:143](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiTrashbin/trashbinSharingToShares.feature#L143)
- [coreApiTrashbin/trashbinSharingToShares.feature:144](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiTrashbin/trashbinSharingToShares.feature#L144)
- [coreApiTrashbin/trashbinSharingToShares.feature:202](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiTrashbin/trashbinSharingToShares.feature#L202)
- [coreApiTrashbin/trashbinSharingToShares.feature:203](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiTrashbin/trashbinSharingToShares.feature#L203)

### Other

API, search, favorites, config, capabilities, not existing endpoints, CORS and others

#### [sending MKCOL requests to another or non-existing user's webDav endpoints as normal user should return 404](https://github.com/owncloud/ocis/issues/5049)

_ocdav: api compatibility, return correct status code_

- [coreApiAuth/webDavMKCOLAuth.feature:42](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiAuth/webDavMKCOLAuth.feature#L42)
- [coreApiAuth/webDavMKCOLAuth.feature:53](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiAuth/webDavMKCOLAuth.feature#L53)

#### [trying to lock file of another user gives http 500](https://github.com/owncloud/ocis/issues/2176)

- [coreApiAuth/webDavLOCKAuth.feature:46](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiAuth/webDavLOCKAuth.feature#L46)
- [coreApiAuth/webDavLOCKAuth.feature:58](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiAuth/webDavLOCKAuth.feature#L58)

#### [Support for favorites](https://github.com/owncloud/ocis/issues/1228)

- [coreApiFavorites/favorites.feature:101](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiFavorites/favorites.feature#L101)
- [coreApiFavorites/favorites.feature:102](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiFavorites/favorites.feature#L102)
- [coreApiFavorites/favorites.feature:103](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiFavorites/favorites.feature#L103)
- [coreApiFavorites/favorites.feature:124](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiFavorites/favorites.feature#L124)
- [coreApiFavorites/favorites.feature:125](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiFavorites/favorites.feature#L125)
- [coreApiFavorites/favorites.feature:126](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiFavorites/favorites.feature#L126)
- [coreApiFavorites/favorites.feature:189](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiFavorites/favorites.feature#L189)
- [coreApiFavorites/favorites.feature:190](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiFavorites/favorites.feature#L190)
- [coreApiFavorites/favorites.feature:191](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiFavorites/favorites.feature#L191)
- [coreApiFavorites/favorites.feature:145](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiFavorites/favorites.feature#L145)
- [coreApiFavorites/favorites.feature:146](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiFavorites/favorites.feature#L146)
- [coreApiFavorites/favorites.feature:147](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiFavorites/favorites.feature#L147)
- [coreApiFavorites/favorites.feature:174](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiFavorites/favorites.feature#L174)
- [coreApiFavorites/favorites.feature:175](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiFavorites/favorites.feature#L175)
- [coreApiFavorites/favorites.feature:176](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiFavorites/favorites.feature#L176)
- [coreApiFavorites/favoritesSharingToShares.feature:91](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiFavorites/favoritesSharingToShares.feature#L91)
- [coreApiFavorites/favoritesSharingToShares.feature:92](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiFavorites/favoritesSharingToShares.feature#L92)
- [coreApiFavorites/favoritesSharingToShares.feature:93](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiFavorites/favoritesSharingToShares.feature#L93)

#### [WWW-Authenticate header for unauthenticated requests is not clear](https://github.com/owncloud/ocis/issues/2285)

- [coreApiWebdavOperations/refuseAccess.feature:21](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavOperations/refuseAccess.feature#L21)
- [coreApiWebdavOperations/refuseAccess.feature:22](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavOperations/refuseAccess.feature#L22)

#### [PATCH request for TUS upload with wrong checksum gives incorrect response](https://github.com/owncloud/ocis/issues/1755)

- [coreApiWebdavUploadTUS/checksums.feature:74](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L74)
- [coreApiWebdavUploadTUS/checksums.feature:75](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L75)
- [coreApiWebdavUploadTUS/checksums.feature:76](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L76)
- [coreApiWebdavUploadTUS/checksums.feature:77](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L77)
- [coreApiWebdavUploadTUS/checksums.feature:79](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L79)
- [coreApiWebdavUploadTUS/checksums.feature:78](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L78)
- [coreApiWebdavUploadTUS/checksums.feature:147](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L147)
- [coreApiWebdavUploadTUS/checksums.feature:148](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L148)
- [coreApiWebdavUploadTUS/checksums.feature:149](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L149)
- [coreApiWebdavUploadTUS/checksums.feature:192](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L192)
- [coreApiWebdavUploadTUS/checksums.feature:193](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L193)
- [coreApiWebdavUploadTUS/checksums.feature:194](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L194)
- [coreApiWebdavUploadTUS/checksums.feature:195](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L195)
- [coreApiWebdavUploadTUS/checksums.feature:196](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L196)
- [coreApiWebdavUploadTUS/checksums.feature:197](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L197)
- [coreApiWebdavUploadTUS/checksums.feature:240](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L240)
- [coreApiWebdavUploadTUS/checksums.feature:241](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L241)
- [coreApiWebdavUploadTUS/checksums.feature:242](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L242)
- [coreApiWebdavUploadTUS/checksums.feature:243](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L243)
- [coreApiWebdavUploadTUS/checksums.feature:244](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L244)
- [coreApiWebdavUploadTUS/checksums.feature:245](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L245)
- [coreApiWebdavUploadTUS/uploadToShare.feature:255](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/uploadToShare.feature#L255)
- [coreApiWebdavUploadTUS/uploadToShare.feature:256](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/uploadToShare.feature#L256)
- [coreApiWebdavUploadTUS/uploadToShare.feature:279](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/uploadToShare.feature#L279)
- [coreApiWebdavUploadTUS/uploadToShare.feature:280](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/uploadToShare.feature#L280)
- [coreApiWebdavUploadTUS/uploadToShare.feature:375](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/uploadToShare.feature#L375)
- [coreApiWebdavUploadTUS/uploadToShare.feature:376](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/uploadToShare.feature#L376)

#### [Renaming resource to banned name is allowed in spaces webdav](https://github.com/owncloud/ocis/issues/3099)

- [coreApiWebdavMove2/moveFile.feature:143](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavMove2/moveFile.feature#L143)
- [coreApiWebdavMove1/moveFolder.feature:36](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavMove1/moveFolder.feature#L36)
- [coreApiWebdavMove1/moveFolder.feature:50](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavMove1/moveFolder.feature#L50)
- [coreApiWebdavMove1/moveFolder.feature:64](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavMove1/moveFolder.feature#L64)

#### [Trying to delete other user's trashbin item returns 409 for spaces path instead of 404](https://github.com/owncloud/ocis/issues/9791)

- [coreApiTrashbin/trashbinDelete.feature:92](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiTrashbin/trashbinDelete.feature#L92)

#### [MOVE a file into same folder with same name returns 404 instead of 403](https://github.com/owncloud/ocis/issues/1976)

- [coreApiWebdavMove2/moveFile.feature:100](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavMove2/moveFile.feature#L100)
- [coreApiWebdavMove2/moveFile.feature:101](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavMove2/moveFile.feature#L101)
- [coreApiWebdavMove2/moveFile.feature:102](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavMove2/moveFile.feature#L102)
- [coreApiWebdavMove1/moveFolder.feature:217](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavMove1/moveFolder.feature#L217)
- [coreApiWebdavMove1/moveFolder.feature:218](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavMove1/moveFolder.feature#L218)
- [coreApiWebdavMove1/moveFolder.feature:219](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavMove1/moveFolder.feature#L219)
- [coreApiWebdavMove2/moveShareOnOpencloud.feature:334](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavMove2/moveShareOnOpencloud.feature#L334)
- [coreApiWebdavMove2/moveShareOnOpencloud.feature:337](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavMove2/moveShareOnOpencloud.feature#L337)
- [coreApiWebdavMove2/moveShareOnOpencloud.feature:340](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavMove2/moveShareOnOpencloud.feature#L340)

#### [COPY file/folder to same name is possible (but 500 code error for folder with spaces path)](https://github.com/owncloud/ocis/issues/8711)

- [coreApiSharePublicLink2/copyFromPublicLink.feature:198](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiSharePublicLink2/copyFromPublicLink.feature#L198)
- [coreApiWebdavProperties/copyFile.feature:1094](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavProperties/copyFile.feature#L1094)
- [coreApiWebdavProperties/copyFile.feature:1095](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavProperties/copyFile.feature#L1095)
- [coreApiWebdavProperties/copyFile.feature:1096](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavProperties/copyFile.feature#L1096)

#### [Trying to restore personal file to file of share received folder returns 403 but the share file is deleted (new dav path)](https://github.com/owncloud/ocis/issues/10356)

- [coreApiTrashbin/trashbinSharingToShares.feature:277](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiTrashbin/trashbinSharingToShares.feature#L277)

#### [Preview. UTF characters do not display on prievew](https://github.com/opencloud-eu/opencloud/issues/1451)

- [coreApiWebdavPreviews/previews.feature:249](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavPreviews/previews.feature#L249)
- [coreApiWebdavPreviews/previews.feature:250](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavPreviews/previews.feature#L250)
- [coreApiWebdavPreviews/previews.feature:251](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavPreviews/previews.feature#L251)

#### [Preview of text file truncated](https://github.com/opencloud-eu/opencloud/issues/1452)

- [coreApiWebdavPreviews/previews.feature:263](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavPreviews/previews.feature#L263)
- [coreApiWebdavPreviews/previews.feature:264](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavPreviews/previews.feature#L264)
- [coreApiWebdavPreviews/previews.feature:265](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavPreviews/previews.feature#L265)

### Won't fix

Not everything needs to be implemented for opencloud.

- _Blacklisted ignored files are no longer required because opencloud can handle `.htaccess` files without security implications introduced by serving user provided files with apache._

Note: always have an empty line at the end of this file.
The bash script that processes this file requires that the last line has a newline on the end.
