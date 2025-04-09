---
title: MongoDB Oplog 和 Journal File
description: 紀錄學習的 MongoDB Oplog 和 Journal File
slug: mongodb-oplog-journalFile
date: 2020-07-20T12:00:00+08:00
categories:
   - MongoDB
weight: 1  
---
## Journaling

關於 journal 在 MongoDB 官方文檔的開頭有以下描述：

> To provide durability in the event of a failure, MongoDB uses *write ahead logging*
 to on-disk [journal](https://www.mongodb.com/docs/v4.2/reference/glossary/#term-journal) files.

可以讓我們了解 MongoDB 的 journal files 是一種預寫日誌(WAL)，作用就類似 MySQL 的 redolog ，都是為了在 DB 服務意外 crash 後恢復數據保證數據持久性的方法。

### Journaling 和 WiredTiger 儲存引擎

WiredTiger 使用 checkpoints 來

恢復的流程大致如下：

1. 在 data files 中找到最後一個 checkpoint
2. 在 journal files 中找到上一步驟中的 checkpoint
3. 應用 journal files 中上一步驟 checkpoint 之後的操作

### Journaling Process

WiredTiger 會為每個 clinet 端發起的寫操作創建一個 journal record 紀錄包含內部的所有寫入操作，例如：執行了 update 操作，journal record 除了會記錄更新操作同時也會記錄相應索引的修改。

MongoDB 設定 WiredTiger 使用內存 buffer 來儲存 journal record

WiredTiger 會在以下情況下將 buffere 中的 journal records 寫入 disk：

- 對於 replica set members (包含 primary 和 secondary)：
    - If there are operations waiting for oplog entries. Operations that can wait for oplog entries include:
        - forward scanning queries against the oplog
        - read operations performed as part of [causally consistent sessions](https://www.mongodb.com/docs/v4.2/core/read-isolation-consistency-recency/#causal-consistency)
    - 對於 secondary 在每次批量應用 oplog 之後。
- 當寫入操作包含 `j: true` 選項。
- 依據 [`storage.journal.commitIntervalMs`](https://www.mongodb.com/docs/v4.2/reference/configuration-options/#storage.journal.commitIntervalMs) 的設置頻率，預設為每 100 ms。
- 當 WiredTiger 創建一個新的 journal file 時，約為每 100MB 數據會創建一個新的 journal file。

<aside>
💡 透過 serverStatus 指令中的 wiredTiger.log 資訊可以查看 WiredTiger journal 的統計資訊。

</aside>

### Journal file

MongoDB 會在 dbPath 設定的目錄下中建立一個名為 journal 的目錄，WiredTiger 的 journal file 會在這個 journal 目錄下：

```bash
➜ ll  /var/lib/mongo/journal
總計 307200
-rw------- 1 root root 104857600  7月 15 17:04 WiredTigerLog.0000000058
-rw------- 1 root root 104857600  7月 12 16:49 WiredTigerPreplog.0000000027
-rw------- 1 root root 104857600  7月 15 16:29 WiredTigerPreplog.0000000054
```

其中 WiredTigerLog.{序號} 是已記錄或使用中的 Journal file，而 WiredTigerPreplog.{序號} 是預先分配的 Journal file。

WiredTiger 的 journal file 最大大小為 100MB，當超過時會建立一個新的 journal file，此外會自動刪除舊的 journal file 僅保留從上一個 checkpoint 恢復所需要的文件。

## Journal record

- WiredTiger 會為每個 clinet 端發起的寫操作創建一個 journal record 紀錄包含內部的所有寫入操作，例如：執行了 update 操作，journal record 除了會記錄更新操作同時也會記錄相應索引的修改。
- 每個 record 會有一個 unique  identifier
- WiredTiger 的 journal record 最小有 128 bytes 的大小。

預設情況下 MongoDB 會將 WiredTiger 超過 128 bytes 的 journal record 使用 `snappy` 進行壓縮，這部分可以透過[`storage.wiredTiger.engineConfig.journalCompressor`](https://www.mongodb.com/docs/v4.2/reference/configuration-options/#storage.wiredTiger.engineConfig.journalCompressor) 設定不同的壓縮演算法

## OpLog

MongoDB 在 primary node 上應用資料庫操作之後會將其記錄到 OpLog，之後 secondary node 會複製並應用這些操作，也就是類似於 MySQL 的 binlog。

oplog 中的每個操作都是冪等，也就是說  oplog 無論在目標 node 上應用一次或多次都會產生相同的結果。

cluster 中的所有 node 都包含 [local.oplog.rs](http://local.oplog.rs) collection 中的 oplog 副本，所以所有的 secondary node 可以向 cluster 內的任意 node 獲取 oplog。

## 參考

Journal file

[Journaling — MongoDB Manual](https://www.mongodb.com/docs/v4.2/core/journaling/)

[WiredTiger Storage Engine — MongoDB Manual](https://www.mongodb.com/docs/v4.2/core/wiredtiger/#storage-wiredtiger-checkpoints)

[【MongoDB】数据存储（Storage）之 日志（Journaling）_奇斯的博客-CSDN博客_wiredtiger](https://blog.csdn.net/chechengtao/article/details/105913943)

OpLOG

[Replica Set Oplog — MongoDB Manual](https://www.mongodb.com/docs/manual/core/replica-set-oplog/)