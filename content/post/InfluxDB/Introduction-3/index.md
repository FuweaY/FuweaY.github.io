---
title: InfluxDB 簡介 - 3
description: 介紹 InfluxDB 儲存引擎 TSM
slug: influxdb-introduction-3
date: 2021-01-18T12:00:00+08:00
categories:
   - InfluxDB
weight: 1  
---
InfluxDB 儲存引擎是 `TSM` 主要是根據 `LSM` 優化而來的

## 目錄與檔案結構

InfluxDB 的資料夾下，大略是以下結構：

```bash
/var/lib/influxdb
|-- data
|   |-- test
|       |-- _series
|       |   |-- 00
|       |   |   `-- 0000
|       `-- autogen
|           `-- 140
|               |-- 000000002-000000003.tsm
|               `-- fields.idx
|-- meta
|   `-- meta.db
`-- wal
    |-- test
        `-- autogen
            `-- 140
                `-- _00004.wal
```

每一個 `shard(如：140)` 都會有一個自己的資料夾，資料夾下包含以下資料：

### data

- `#####.tsm`：實際儲存數據的文件。
- `#####.tombstone`：紀錄刪除的數據，當 `TSM Files` 重寫壓縮刪除了數據後，會將此檔案刪除。
- `fields.idx`：存放 `fields` 的 metadata。

### meta

- `meta.db`：用來儲存 `InfluxDB` 的 `metadata`，包括 `users`、`databases` 、 `retention policies`、`shards` 和 `continuous queries`。

### wal

- `#####.wal`：數據寫入時，會先 `appended` 到 `WAL` 文件避免數據丟失。 當 `Compact` 到 `TSM Files` 後，會刪除對應的 `WAL` 文件釋放空間。

## TSM 組件

### WAL (Write Ahead Log)

一種專門針對寫入優化的儲存格式，寫入時是將資料 `append` 到 `WAL File`，透過順序寫入 Disk 的方式，大幅提升數據寫入的性能 ，但也因此犧牲了讀取的性能。

寫入資料時，除了寫入 `Cache` 還會寫入 `WAL` 確保數據不會丟失，用途類似於 MySQL 的 `binlog` 。

WAL 被儲存成 `_00001.wal` 這種樣子的檔案，每當一個檔案到達 `10MB` 之後會再創建一個新的 WAL 文件並遞增編號，如： `_00002.wal`。

### Cache

寫入資料時，除了寫入 `WAL` 還會寫入 `Cache`， `cache` 是 `WAL` 文件中的數據在內存中的緩存，因此可以透過重新讀取 `WAL` 文件，重新創建 `cache` 緩存。

在查詢數據時，會將 `Cache` 和 `TSM Files` 的數據進行合併，在 `Cache` 中的數據優先度較高。

和 `TSM Files` 不同，緩存在 `cache` 內的數據不會被壓縮。

`cache` 有 2 個很重要的設定，每次寫入數據時都會檢查以下閾值：

1. 當 `cache` 內的資料達到 `cache-snapshot-memory-size(預設25M)` 所設定的大小時，會將數據寫入 `TSM Files` 並釋放內存空間。
2. 當 `cache` 內的資料超過 `cache-max-memory-size(預設1G)` 設定的大小時，則會拒絕數據寫入。

此外，還有一個設定－ `cache-snapshot-write-cold-duration(預設10m)`，當該 `shard` 超過這段時間沒有收到 `insert、delete` 請求也會將數據寫入 `TSM Files` 釋放內存空間。

### TSM Files

用於存放經過壓縮的數據。

### FileStore

用來管理 `TSM Files`，並可以確保 `TSM Files` 再壓縮重寫時保持 `atomically(原子性)`。

### Compactor

負責將 `TSM Files` 進行優化，便於查詢和節省空間，透過以下 4 種方式：

- Snapshots：將 `Cache` 和 `WAL` 中的數據轉換為 `TSM Files`，來釋放 `WAL segment` 和 `Cache` 所使用的內存和磁碟空間。
- Level Compactions：分為 1~4 級，會將 `TSM Files` 存 `Snapshots` 壓縮到 1級文件，再將多個 1級文件壓縮成 2級文件，直到文件達到 4級或者達到 `TSM Files` 的最大大小限。低級別的文件表示數據時間點較新，為了避免解壓縮耗費大量 CPU，因此使用較低的壓縮率。高級別的文件表示數據時間點較舊，因為使用的頻率降低，所以使用較高的壓縮率。
- Index Optimization：當積累了許多 4 級 `TSM Files` 時， 內部索引越來越大，造成搜尋的成本增加，透過將相同 `series` 的 `points` 拆分到一組新的 `TSM Files`，讓同一個 `series` ，避免同一個 `series` 的資料需要跨越多個 `TSM Files` 來查詢。
- Full Compaction：當該 `shard` 已經變成冷數據，或者 `shard` 上出現了 `delete` 操作 ，會執行包括 `Level Compactions` 和 `Index Optimization` 的所有優化，以此產生最佳的 `TSM Files`，除非該 `shard` 有心的 `insert、delete` ，否則不會再執行其他 `compaction`。

### Compaction Planner

決定哪些 `TSM Files` 可以進行壓縮，並確保多個併發的 `Compactor` 不會互相影響。

### Compression

根據不同的資料型態對資料進行壓縮和解碼。

## 資料流

### INSERT

`INSERT` 操作會被 `appended` 到當前的 `WAL segment`，並且也會被寫到 `cache` 中。

- 當 `WAL segment` 達到 `10MB` 之後，會關閉並建立一個新的 `WAL segment`。
- 當 `WAL segment` 被關閉後，則會執行 `Snapshots` 將資料寫入 `TSM Files` 並且 `fsync` 到硬碟上，隨後讓 `FilsStore` 進行加載和引用。
- 當 `cache` 內緩存的數據量達到 `cache-snapshot-memory-size` 時，也會啟動 `Snapshots`。
- 當 `cache` 內緩存的數據量進一步達到 `cache-max-memory-size` 時，則會拒絕寫入直到 `Snapshots` 線程將 `cache` 釋放。

### UPDATE

針對已經寫入的 `Points` 寫入新的值，在 InfluxDB 中等同於 `INSERT` 操作，較新寫入的值具有優先權會覆蓋舊的值。

### DELETE

`DELETE` 在 `WAL segment` 寫入之後，會更新 `Cache` 和 `FileStore` 來進行刪除。

- 刪除 `Cache` 內的相關數據。
- `FileStore` 會在 `TSM` 資料夾下建立 `.tombstone` 檔案，當從 `TSM Files` 查詢數據時會比對 `.tombston` 的數據來過濾刪除的數據。
- 當觸發 `Compactor` 重寫壓縮 `TSM Files` 時，會透過 `.tombstone` 讓刪除的數據不再寫入新的 `TSM Files`，達成真正刪除數據的動作。

### SELECT

`SELECT` 操作透過 `FileStore` 和 `Cache` 讀取資料。

- `Cache` 中的值因為較新，所以會 `overlaid(覆蓋)` 在 `FileStore` 返回的值上。

## 參考

[In-memory indexing and the Time-Structured Merge Tree (TSM) - influxdata 文檔](https://docs.influxdata.com/influxdb/v1.8/concepts/storage_engine/)

[influxdb/tsdb/engine/tsm1/DESIGN.md - github](https://github.com/influxdata/influxdb/blob/1.8/tsdb/engine/tsm1/DESIGN.md)

[LSM Tree 学习笔记 - fatedier blog](http://blog.fatedier.com/2016/06/15/learn-lsm-tree/)

[InfluxDB详解之TSM存储引擎解析（一） - fatedier blog](http://blog.fatedier.com/2016/08/05/detailed-in-influxdb-tsm-storage-engine-one/)

[InfluxDB详解之TSM存储引擎解析（二） - fatedier blog](http://blog.fatedier.com/2016/08/15/detailed-in-influxdb-tsm-storage-engine-two/)