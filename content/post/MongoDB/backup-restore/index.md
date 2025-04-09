---
title: MongoDB 備份還原
description: 紀錄學習的 MongoDB 備份還原
slug: mongodb-backup-restore
date: 2020-07-15T12:00:00+08:00
categories:
   - MongoDB
weight: 1  
---
## mongodump、mongorestore

將 MongoDB 的資料內容導出成二進制檔案

```bash
mongodump <options> <connection-string>
mongorestore <options> <connection-string> <directory or file to restore>
```

可以使用以下 `--uri` 或者是 `--host` 的方式指定要連線的 MongoDB 服務，如下範例：

```bash
# 連線到 mongoDB instance
mongodump --uri="mongodb://mongodb0.example.com:27017" [additional options]
mongodump --host="mongodb0.example.com:27017"  [additional options]
mongodump --host="mongodb0.example.com" --port=27017 [additional options]

# 連線到 mongoDB replica set (優先從 primary 讀取)
mongodump --uri="mongodb://mongodb0.example.com:27017,mongodb1.example.com:27017,mongodb2.example.com:27017/?replicaSet=myReplicaSetName" [additional options]
mongodump --host="myReplicaSetName/mongodb0.example.com:27017,mongodb1.example.com:27017,mongodb2.example.com" [additional options]

# 連線到 mongoDB replica set (優先從 secondary 讀取)
mongodump --uri="mongodb://mongodb0.example.com:27017,mongodb1.example.com:27017,mongodb2.example.com:27017/?replicaSet=myReplicaSetName&readPreference=secondary" [additional options]
mongodump --host="myReplicaSetName/mongodb0.example.com:27017,mongodb1.example.com:27017,mongodb2.example.com:27017" --readPreference=secondary [additional options]

# 連線到 mongoDB replica set (優先從 secondary 讀取，並指定 tag)
mongodump --uri="mongodb://mongodb0.example.com:27017,mongodb1.example.com:27017,mongodb2.example.com:27017/?replicaSet=myReplicaSetName&readPreference=secondary&readPreferenceTags=region:east" [additional options]
mongodump --host="myReplicaSetName/mongodb0.example.com:27017,mongodb1.example.com:27017,mongodb2.example.com:27017" --readPreference='{mode: "secondary", tagSets: [ { "region": "east" } ]}' [additional options]
```

### 行為

1. 只能還原到相同版本的 MongoDB 服務，並且 mongodump 和 mongorestore 版本也需要相同。
2. 預設情況下會優先連線 Primary node，可以透過 `readPreference` 選項來調整。
3. mongodump 不會 dump `local` database。
4. mongodump 不會備份 index，需要自行重建。
5. 如果有使用 read-only views，mongodump 預設只會 dump 其 metadata，需要添加 `--viewAsCollections` 來匯出 view 中的數據。
6. mongodump 會在以下情況下 fails：
    - 當 resharding 正在運行時。
    - 當 reshardCollection 指令在 mongodump 操作期間執行時。
7. 對 WiredTiger 引擎使用 mongodump 會匯出未壓縮的數據。
8. mongodump 會影響到 mongod 的性能，如果數據量大於系統內存 mongodump 會導致 working set 被推出內存。
9. mongodump 需要具有欲備份 database 的 find 權限，內建的 `backup`  role 具有備份所有 database 的權限。

### 常用選項

- `--uri` ：

    ```bash
    --uri="mongodb://[username:password@]host1[:port1][,host2[:port2],...[,hostN[:portN]]][/[database][?options]]"
    
    mongodump --username joe --password secret1 mongodb://mongodb0.example.com:27017 --ssl
    ```

- `--host=<hostname><:port>, -h=<hostname><:port>`：指定要連線的 mongod。
- `--readPreference=<string|document>`：指定優先讀取的 node，預設值為 primary，範例如下：
    - maxStalenessSeconds：由於各種原因 secondary node 可以會落後 prmary node，該選項用來設定 secondary node 最大延遲的秒數，當 secondary 落後超過該秒數則 mongodump 停止進行讀取操作，該值必須大於等於 90。

    ```bash
    # 優先讀取 secondary node
    --readPreference=secondary
    
    # 優先讀取 secondary 中 tag 為 region:east 的 node，允許的最大延遲為 120
    --readPreference='{mode: "secondary", tagSets: [ { "region": "east" } ], maxStalenessSeconds: 120}'
    ```

- `--db=<database>, -d=<database>`：指定要備份的 database，若未指定會備份 `local` 以外的所有 database。
- `--collection=<collection>, -c=<collection>`：指定要備份的 collection，若未指定會備份 database 內的所有 collection。
- `--query=<json>, -q=<json>`：必須搭配 `--collection` 選項，僅匯出符合此條件的數據，必須使用單引號 `'` 將查詢文檔包起來，範例如下：

    ```bash
    mongodump -d=test -c=records -q='{ "a": { "$gte": 3 }, "date": { "$lt": { "$date": "2016-01-01T00:00:00.000Z" } } }'
    ```

- `--gzip`：壓縮輸出。
- `--oplog`：當沒有該選項時，mongodump 運行期間的寫入

  將 mongodump 運作期間產生的 oplog 一併匯出，該檔案提供了

  mongodump --oplog 運行過程中 client 端運行以下指令將導致 mogodump 失敗：

    - renameCollection
    - db.collection.renameCollection()
    - db.collection.aggregate() with $out

  mongodump --oplog 必須完整備份 replica set ，因此不可和 `--db` 及 `--collection` 一起使用。


### 備份策略

mogodump、mongorestore 可以作為單實例、一般 cluster 的備援方案

mongodump、mongorestore 通過與正在運行的 mongod instance 交互來運行，不僅會產生流量，還會強制資料庫透過記憶體讀取所有數據，因此會導致mongod 性能下降。

mogodump、mongorestore 不能作為  sharding cluster 的備援方案，因為 mongodump 創建的備份不會維護跨 shard 事務的原子性保證。

對於 sharding cluster 建議使用以下支持維護跨 shard 事務原子性保證的備援方案：

- [MongoDB Atlas](https://www.mongodb.com/cloud/atlas?tck=docs_databasetools)
- [MongoDB Cloud Manager](https://www.mongodb.com/cloud/cloud-manager?tck=docs_databasetools)
- [MongoDB Ops Manager](https://www.mongodb.com/products/ops-manager?tck=docs_databasetools)

## mongoexport

可以將 MongoDB 中的數據匯出成 JSON 或 CSV 格式。

```bash
mongoexport --collection=<coll> <options> <connection-string>
```

## mongoimport

將 mongoexport 或其他第三方匯出的 JSON、CSV 或 TSV 格式的數據匯入到 MongoDB。

## 參考

[mongodump — MongoDB Database Tools](https://www.mongodb.com/docs/database-tools/mongodump/)

[mongorestore — MongoDB Database Tools](https://www.mongodb.com/docs/database-tools/mongorestore/)

[mongoexport — MongoDB Database Tools](https://www.mongodb.com/docs/database-tools/mongoexport/)

[mongoimport — MongoDB Database Tools](https://www.mongodb.com/docs/database-tools/mongoimport/)

[Back Up and Restore with MongoDB Tools — MongoDB Manual](https://www.mongodb.com/docs/manual/tutorial/backup-and-restore-tools/)

[MongoDB Backup Methods — MongoDB Manual](https://www.mongodb.com/docs/manual/core/backups/)

[MongoDB的常规备份策略 - yaoxing - 博客园 (cnblogs.com)](https://www.cnblogs.com/yaoxing/p/mongodb-backup-rules.html)