---
title: PostgreSQL Replication
description: 紀錄學習的 PostgreSQL Replication
slug: Postgre-replication
date: 2020-02-19T12:00:00+08:00
categories:
   - PostgreSQL
weight: 1  
---
Replication 是提供高可用及 load balance 的基礎，因此不論在哪一種 database 都是一項重要的功能。

在不同的 Database 下對於主備 Server 都有不同的術語：

|  | MySQL < 8.0.26 | MySQL > 8.0.26 | MongoDB | PostgreSQL |
| --- | --- | --- | --- | --- |
| 主 | Master | Source | Primary | Primary |
| 備 | Slave | Replica | Secondary | Standby |

在 MySQL 中透過 Server 層的 binlog 透過

在 PG 中則是直接透過 WAL (相對於 InnoDB redo log)

## Standby (Slave) 的行為

當 server 啟動時數據目錄中包含 `standby.signal` 文件時，則 server 會進入 standby 模式。

在 standby 模式下，standby 可以透過以下兩種方式來獲取並回放 Primary 上的 WAL：

- WAL Archive
- Streaming Replication

除此之外 Standby 還會嘗試恢復在 pg_wal 目錄中的任何 WAL。

Standby 在啟動時會有以下行為：

1. 調用 `restore_command` 進行恢復。
2. 當步驟1 達到 WAL 末尾且恢復失敗，則將嘗試恢復 pg_wal 目錄中可用的 WAL。
3. 當步驟2失敗，且配置了 Streaming Replication ，則會嘗試連接到 Primary 並從存檔或 pg_wal 目錄中找到有效的紀錄開始 Streaming WAL。
4. 當步驟3失敗，或是沒有配置  Streaming Replication 則將回到步驟1重新嘗試

以上步驟的 retry 循環或一直持續到 server 關閉或者 failover。

failover 可以透過運行 `pg_ctl promote`、調用 `pg_promote()` 或發現 trigger file (promote_trigger_file) 時，則會退退出 standby 模式成為 primary 提供正常操作。

## 建議

- 在不同的主要版本號中通常是無法進行 log shipping 的
- 雖然 PG 沒有正式支持不同次要版本之間的 log shipping，但在次要版本中一般不更改 Disk 格式，並且新的次要版本更可能從舊的次要版本讀取 WAL，因此升級小版本號時建議先升級 standby。
-
- 在 Primary 上設置 Continuous archiving 時，應該考慮將歸檔位置設為 standby 本身或是非 Primary 機器上，避免因為 Primary 機器本身出問題而無法訪問。
- 使用 Streaming Replicaiton 需要注意以下事項：
    - 創建 replication role，並在 ph_hba.conf 有正確的設置，確保 standby 能夠連線 Primary。
    - 確保 Primary 上的 `max_wal_senders` 設置足夠大。
    - 如果使用 Replication Slot 還需要確保 `max_replication_slots` 設置。

## 設置 Standby

### Log Shipping (warm standby)

在 PG 9.0 之前，PostgreSQL 只提供這種異步 replication 方式，這是透過 Primary 每次傳送一個 WAL 給 Standby 來實現，缺陷也很明顯就是 Standby 至少會落後 Primary 一個 WAL。

#### 配置方式

1. 在 Primary 中新增以下設置

    ```sql
    archive_mode = on
    archive_command = 'test ! -f /var/data/xlog_archive/%f && cp %p /var/data/xlog_archive/%f'
    ```

2. 對 Primary 進行 Base backup

    ```bash
    pg_basebackup -P -D /var/data/backup/pg_back_$(date +"%F_%T")
    ```

3. 還原到 Standby

    ```sql
    cp -R ./backup/pg_back_2023-04-17_07\:06\:57/* /var/data/postgres
    ```

4. 修改 Standby 設置，並以 standby 模式運行

    ```sql
    restore_command = 'cp /var/data/xlog_archive/%f "%p"'
    ```

    ```sql
    touch standby.signal
    pg_ctl -D /var/data/postgres -l logfile start
    ```

5. 透過在 Primary 新增資料，並使用 `pg_switch_wal()` 強制觸發 WAL 的切換來觀察
    - primary

        ```sql
        -- primary
        [postgres @ postgres] [local]:5432] 07:16:07 > \c test
        You are now connected to database "test" as user "postgres".
        [postgres @ test] [local]:5432] 07:16:08 > select * from test;
         id | name
        ----+-------
          1 | test1
        (1 row)
        
        Time: 1.607 ms
        [postgres @ test] [local]:5432] 07:16:13 > insert into test values(2,'test2');
        INSERT 0 1
        Time: 1.594 ms
        [postgres @ test] [local]:5432] 07:16:24 > select * from test;
         id | name
        ----+-------
          1 | test1
          2 | test2
        (2 rows)
        
        Time: 0.330 ms
        [postgres @ test] [local]:5432] 07:16:25 > SELECT pg_switch_wal();
         pg_switch_wal
        ---------------
         0/30003D8
        (1 row)
        
        Time: 171.419 ms
        ```

    - standby

        ```sql
        [postgres @ postgres] [local]:5432] 07:15:41 > \c test
        You are now connected to database "test" as user "postgres".
        [postgres @ test] [local]:5432] 07:15:44 > \dt
                List of relations
         Schema | Name | Type  |  Owner
        --------+------+-------+----------
         public | test | table | postgres
        (1 row)
        
        [postgres @ test] [local]:5432] 07:15:52 > select * from test;
         id | name
        ----+-------
          1 | test1
        (1 row)
        
        Time: 0.660 ms
        [postgres @ test] [local]:5432] 07:16:00 > select * from test;
         id | name
        ----+-------
          1 | test1
        (1 row)
        
        Time: 0.425 ms
        
        -- 在 primeary 觸發 WAL 刷新後
        [postgres @ test] [local]:5432] 07:16:27 > select * from test;
         id | name
        ----+-------
          1 | test1
          2 | test2
        (2 rows)
        
        Time: 0.334 ms
        ```


### 相關參數

#### Archiving

- archive_mode (enum)：設置是否進行 archive，有以下三種設置：
    - off：關閉 archive。
    - on：正常模式下開啟 archive，但是在 recovery 或是 standby 時不進行 archvie。\

      當 standby提升為 primary 時只會 archive 自己產生的 WAL，而不會 archive 原本從另一個 primary 取得的 WAL。

    - always：不論什麼模式都進行 archive。
- archive_command (string)：設置 archive WAL 的 shell 指令。
- archive_library (string)：設置用於 archive WAL 的 library，若為空表示使用 `archive_command`。
- archive_timeout (integer)：此參數為時間設置預設單位為秒，當此參數大於 0 時，會強制 server 只要距離上次切換到新的 WAL 已經過去這段時間，且有數據庫活動就必須切換到新的 WAL。

  因為 archive 只會針對完成的 WAL 作用，因此設置此參數可以避免當 WAL 輪換不頻繁時，導致 Transaction 過久沒有被安全的歸檔或者同步到 standby。

  注意：強制切換 WAL 的文件大小和完整的文件大小相同都是 16MB，因此設置太短的 `archive_timeout` 會導致歸檔存檔膨脹，通常 1 分鐘左右的設置是合理。如果希望 Primary 和 Standby 之間的延遲更小應該考慮 streaming replication 而不是調整此值。


#### Archive Recovery

此處用於設置 recovery 期間的設定，recovery 包含以下兩者：

- recovery mode：透過在數據目錄中創建 `recovery.signal` 設置
- standby mode：透過在數據目錄中創建 `standby.signal` 設置，首先進入 recovery mode 並在恢復到歸檔的 WAL 末尾時不會停止，而是會嘗試透過 `primary_conninfo` 向指定的 primary 請求 WAL 或 `restore_command` 來繼續回放 WAL。

備注：如果 2 個 signal 都被建立則會以 standby mode 優先。

- restore_command (string)：用於設置如何將歸檔的 WAL 用於 restore 的 shell 命令。

  範例如下：

    ```sql
    restore_command = 'cp /mnt/server/archivedir/%f "%p"'
    ```

  上述設定表示將複製 `/mnt/server/archivdir` 底下的 WAL 到 pg_wal 中。

    - %f：表示歸檔 WAL 文件名稱。
    - %p：表示放置用於恢復 WAL 的目錄 (pg_wal)。
- archive_cleanup_cpmmand (string)：用於設置清理 standby 不再需要的歸檔 WAL shell 命令。

  範例如下：

    ```sql
    archive_cleanup_command = 'pg_archivecleanup /mnt/server/archivedir %r'
    ```

  上述設定表示運行 `pg_archivecleanup` 將上一個 WAL 文件刪除。

    - %r：表示需要的 WAL 文件的上一個 WAL 文件名稱。
- recovery_end_command (string)：用於設置當 recovery 結束時執行的 shell 命令，例如可以用於發送 email 通知：

    ```sql
    recovery_end_command = 'echo "Recovery complete" | mail -s "Recovery complete" admin@example.com'
    ```


### 小節

Log Shipping 是一種異步且即時性不高的 replication 方式，但是他相應的優勢就是因為是透過 primary 歸檔後的 WAL 進行恢復，因此不會有 Streaming Replicaion 丟失 WAL 導致無法復原的問題。

雖然不推薦將 Log Shipping 做為主力的 replication 方案，但是可以將其同時和 Streaming Replication 一起運行作為附屬方案，只有當 Streaming Replication 因為某些原因跟不上 Primary 導致對應的 WAL 被刪除時，透過歸檔的 WAL 來進行復原。

## Streaming Replication (流複製 或稱 物理複製)

在 PG 9.0 開始提供了 Steaming  Replication，雖然和 Log Shipping 一樣都是透過 WAL 日誌的回放來達到 Primary 和 Standby 的同步，但是 Streaming Replication 只要在 WAL 中一產生變化就會發送給 Standby，而不需要等到 WAL 切換才傳送，也就是說 Streaming Replication 會有更短的 replication delay。

具體實現方面大體上其實和 MySQL 差不多，只是 MySQL 是透過 binlog 中的邏輯複製，但 PG 的 WAL 比較接近 MySQL redo log 也就是說是根據實際修改 Disk 物理空間的紀錄，也就是說在 PG 中 Master 和 Slave 的底層數據狀態是完全一致的。

在 Streaming Replication 有以下角色：

- Master 上的 backend 進程：執行收到的 Query 指令，在修改數據前先寫入 WAL，並在 commit 的時候將 WAL fsync 到 Disk。
- Master 上的 WAL sender 進程：將 WAL 發送給 Slave 的  WAL receiver 。
- Slave 上的 WAL receiver 進程：接收並持久化儲存 Master WAL sender 發送的 WAL。可以類比為 MySQL 的 IO THREAD。
- Slave 上的 startup 進程：Apply  WAL receiver  接收到的 WAL。可以類比為 MySQL 的 SQL THREAD。

### 同步 OR 異步

此外和 MySQL 一樣也分為異步或同步，主要透過以下參數控制：

- synchronous_commit (enum)：指定 server 在完成多少 WAL 處理後才返回 Clinet 端成功。

  當 `synchronous_standby_names`  為空時，分為兩種設置：

    - on：此為默認值且 OFF 以外的設置都視為 ON，當 WAL fsync 到 Disk 才返回成功給 Client 端。
    - off：不會等待 WAL 的處理，也就是說可能會丟失 Transaction，最多可能會導致丟失 `wal_write_delay` * 3。

  當 `synchronous_standby_names`  不為空時，分為5種設置：

  總共有以下 5 個值：

    - remote_apply：保證 Slave 已經收到該 Transaction 的 WAL 並調用 fsync() disk，確保了該 Transaction 只有 Master、Slave 都發生數據損壞才會丟失 Transaction，並且還完成了 WAL 回放，因此在 Master、Slave 上擁有一致性讀取。
    - on：也可稱為 remote_flush，保證 Slave 已經收到該 Transaction 的 WAL 並調用 fsync() disk，確保了該 Transaction 只有 Master、Slave 都發生數據損壞才會丟失 Transaction，但因為還沒有回放 WAL，因此 Master、Slave 沒有一致性讀取。
    - remote_write：保證 Slave 已經收到該 Transaction 的 WAL 並寫入 OS 緩存，但未調用 fsync() disk，也就是說 PG server crash 並不會導致 Transaction 丟失，但是整個 OS 系統 Crash 將可能導致 Transaction 丟失。
    - local：等同於 `synchronous_standby_names`  為空時的 ON 設置，也就是說只要 Master 的 WAL 有 fsync 即可不必等待 slave，也就是異步複製。
    - off：不會等待 WAL 的處理，也就是說可能會丟失 Transaction，最多可能會導致丟失 `wal_write_delay` * 3。

  注意：當 `synchronous_standby_names` 為空時，除了 OFF 以外的設定都會被視為 ON。

  ![Untitled](https://s3-us-west-2.amazonaws.com/secure.notion-static.com/b38ca1e7-a55c-4ea4-afb6-a20fc4126332/Untitled.png)

- synchronous_standby_names (string)：指定要求同步 replication 的 slave 列表。

  這邊列出 Slave 在 primary_conninfo 中設置的 application_name (如果沒有則是 cluster_name)。

  可以設置成以下格式：

    ```bash
    [FIRST] num_sync ( standby_name [, ...] )
    ANY num_sync ( standby_name [, ...] )
    standby_name [, ...]
    ```

  其中 num_sync 為數字，表示必須至少有多少 Slave 回覆已同步完成的 Ack 訊號，和 FIRST、ANY 組合有以下效果：

    - [FIRST] num_sync：必須按照列表中的順序前 num_sync 數量的 slave 回覆已同步完成的 Ack 訊號才可以，之後的 slave 作為同步 slave 的備選只有優先序高的出現問題才會替補。
    - ANY num_sync：只要列表中的任意 num_sync 數量的 slave 回覆已同步完成的 Ack 訊號即可。

### 建置範例

1. 檢查 Source 的設定

    ```bash
    listen_addresses = '*'
    
    # 以下皆為預設值
    hot_standby = ON
    wal_level = replica
    max_wal_senders = 10
    ```

2. 設置 replication 用的 User

    ```bash
    createuser -U postgres --replication repl
    ```

3. 確認 Source 中的 pg_hba.conf 是否能接受 Replica 機器的連線

    ```bash
    # TYPE  DATABASE        USER            ADDRESS                 METHOD
    host    replication     repl            172.17.0.3/16           trust
    ```

4. 對 Source 進行 Base backup

    ```bash
    pg_basebackup -P -D /var/data/backup/pg_back_$(date +"%F_%T") -R -U repl
    ```

   -R 選項會在備份檔中有以下不同

    - 生成 standby.signal
    - 在 postgresql.auto.conf  中多出 primary_conninfo 的資訊：

        ```bash
        primary_conninfo = 'user=repl passfile=''/home/postgres/.pgpass'' channel_binding=prefer port=5432 sslmode=prefer sslcompression=0 sslsni=1 ssl_min_protocol_version=TLSv1.2 gssencmode=disable krbsrvname=postgres target_session_attrs=any'
        ```

5. 將 Source 剛剛的 Base backup 還原到 Replica 上

    ```bash
    cp -R /root/pg_data/source/backup/pg_back_2023-04-13_07\:20\:40/* /root/pg_data/replica/postgres
    ```

6. 啟動 replica

    ```bash
    pg_ctl -D /var/data/postgres -l logfile start
    ```

    ```bash
    2023-04-13 09:10:54.724 UTC [1136] LOG:  consistent recovery state reached at 0/1A000000
    2023-04-13 09:10:54.725 UTC [1135] LOG:  database system is ready to accept read-only connections
    2023-04-13 09:10:54.733 UTC [1140] LOG:  started streaming WAL from primary at 0/1A000000 on timeline 1
     done
    ```


### 小節

實務上可以結合 log shipping 和 streaming replication 一起使用，當 streaming replicaiton 因為 TCP 異常或是 Primary 丟棄所需的 WAL 時，PG 能夠自動切換回使用 log shipping 透過 restore_command 繼續追趕 Pimary 的進度，直到趕上 Primary 後 Standby 會重新嘗試切回 streaming repliction。

## Replication Slot

### 功用 - Slave 所需的 WAL 在 Master 上的保留

在 PostgreSQL 9.4 之前是沒有 Replication Slot 的，Slave 因為某些原因暫停 replication 之後，因為在 Master 上需要應用的 WAL 已經被清除，導致 Slave 恢復 replication 後發生以下狀況：`requested WAL segment 0000000100000000000000xxx has already been removed`，為了避免這個狀況可以透過以下參數的配置：

- wal_keep_size：指定 pg_wal 目錄下至少需保留的 WAL 大小。

  默認值為 0 表示不會額外保留 WAL，也就是說 slave 可用的 WAL segments 數量取決於前一個 checkpoint 和 WAL archiving 狀態。

  備註：在 PG 13 之前是使用 wal_keep_segments 來控制，他們之間的關係是 wal_keep_size = wal_keep_segments  * wal_segment_size (一般是 16 MB)。

- archive_command：設置 WAL archive 的指令。

  將 WAL archive 到其他地方，這樣 Slave 就能從已經 archive 的 WAL 繼續恢復。


不過以上方法會保留較多的 WAL，因此有了 Replication Slot 用來確保 Master 在所有 Slave 收到 WAL segments 之前不會刪除對應的部分。

但這也帶來了相應的問題：當 Slave 發生異常無法回覆 Master 當前的 lsn 位置，就會導致 Master 不斷保留 WAL 最後導致 Disk 空間用盡而無法提供正常服務。

在 PostgresSQL 13 之前，只能透過監控 replication 狀態來盡早發現進行處理，例如以下語句可以知道保留的 lsn 到當前最新的 redo_lsn 落後多少：

```bash
postgres=# SELECT redo_lsn, slot_name,restart_lsn, round((redo_lsn-restart_lsn)/1024/1024,2) AS  MB_behind  
FROM pg_control_checkpoint(), pg_replication_slots;
 redo_lsn  | slot_name  | restart_lsn | mb_behind 
-----------+------------+-------------+-----------
 0/AAFF2D0 | pgstandby1 | 0/AAFF3B8   |      0.00
(1 row)
```

在 PostgresSQL 13 時推出了 `max_slot_wal_keep_size` 這個設置：

- max_slot_wal_keep_size (integer)：設置 replication slot 在 pg_wal 目錄中保留的最大 wal 大小。

  當一個 slot 的 restart_lsn 落後於 current_lsn 本數值後會停止 stream replication，且該 slot 會被標記為無效，同時之前的 WAL 將被刪除。

  預設值為 -1 表示不限制。


以上參數極大避免了在發生 Master 被 WAL 撐爆的問題，並且在 `pg_replication_slots` 也增加了 `wal_status` 和 `safe_wal_size` 字段方便監控：

```bash
postgres=# select wal_status,safe_wal_size  from pg_replication_slots;
 wal_status | safe_wal_size 
------------+---------------
 reserved   |    1078987728
(1 row)
```

- `wal_status`：表示該 Slot 所需的 WAL 狀態
    - reserved：在 max_wal_size 之內。
    - extended：超出了 max_wal_size，但仍在 `wal_keep_size` 或 `max_slot_wal_keep_size` 的保護範圍內。
    - unreserved：表示 WAL 已經不在保護範圍內，這是一個臨時狀態，隨後可能變成 `extended` 或 `lost`。
    - lost：代表 WAL 已被刪除。
- `safe_wal_size`：只有設置 `max_slot_wal_keep_size` 才會出現，表示還能寫入多少 WAL 大小才不會超過 `max_slot_wal_keep_size` 。當此值 ≤ 0 時，一旦觸發 checkpoint 就會發生 wal_status = lost 的狀況。

### 功用 - 改善 Master 的 vacuum 過早清除 Slave 所需的紀錄

當在 Master 使用 vacuum 指令或者觸發 auto vacuum 時，如果此操作在 Slave 回放時恰巧正在進行相關表的查詢時，會檢測出 vacuum 要清理的元組 (tuple) 仍被使用中，因此不能在 Slave 中立刻清除會在等待 `max_standby_streaming_delay` (默認為 30 秒) 之後終止該查詢操作，並返回以下錯誤後進行 vacuum：

```bash
ERROR: canceling statement due to conflict with recovery
Detail: User query might have needed to see row versions that must be removed
```

在沒有使用 Replication Slot 時，可以透過以下 2 個參數調整：

- 在 Master 上設置 vacuum_defer_cleanup_age (integer)：指定 vacuum 和 Hot updates 將延遲多少 transaction 之前的 dead tuple 將不進行刪除。

  默認值為 0，表示可以盡快刪除 dead tuple。

  問題：調大此值雖然可以多保留 N 個 Transaction 之前的 dead tuple，但這是根據 Master 上的 Transaction 狀況決定，實際上很難預測 sLAVE 需要多少額外的寬限時間，因此不太實用。

- 在 Slave 上設置 hot_standby_feedback (boolean)：當設置為 ON，表示 Slave 會通知 Master 自己目前的最小活躍事務id (xmin) 值，這樣 Master 在執行 vacuum 操作時會暫時不清除大於該 xmin 的 dead tuple。

  問題：當沒有 Slot 時，在 Slave 未連結的任何時間段都不提供保護。


當有 Replication Slot 和 `hot_standby_feedback` 參數配合時，slot 會保持紀錄最後回傳的 xmin 保護 dead tuple。

不過理所當然的事情有一體兩面，如果 Slave 上該查詢運行過久，Master 上可能會發生更嚴重的表膨脹問題。

### 使用 Physical Replication slot

1. 檢查 Source 的設定

    ```bash
    listen_addresses = '*'
    
    # 以下皆為預設值
    hot_standby = ON
    wal_level = replica
    max_wal_senders = 10
    max_replication_slots = 10
    ```

2. 在 Sourrce 上建立 replication slot

    ```bash
    postgres=# SELECT * FROM pg_create_physical_replication_slot('node_a_slot');
      slot_name  | lsn
    -------------+-----
     node_a_slot |
    
    postgres=# SELECT slot_name, slot_type, active FROM pg_replication_slots;
      slot_name  | slot_type | active 
    -------------+-----------+--------
     node_a_slot | physical  | f
    (1 row)
    ```

3. 修改 Replica 上的設定

    ```bash
    primary_conninfo = 'user=repl port=5432 host=172.17.0.2 application_name=replica'
    primary_slot_name = 'node_a_slot'
    ```


### 小節

我認為應該使用 replication slot

## Logical Replication (邏輯複製)

### PG 15(14) 限制

1. 僅會同步 table (包含 partitioned table)，不支持 views, materialized views, or foreign tables。
2. 不會複製 schema 及 DDL 命令。
3. 不複製 sequence data
4.
5. 不支持 Large objects。
6.

### 建置範例

1. 調整 Master、Slave 的設定，其中 `wal_level` 必須調整為 `logical`

    ```bash
    wal_level = logical
    ```

2. 在 Master 建立一個用於 Logical Replication 的 role：

    ```sql
    create user logic_repl with replication;
    
    # 還必須提供 usage schema、select table 的權限
    grant usage on SCHEMA test_schema to logic_repl;
    grant select on test_schema.test_table to logic_repl
    ```

3. 設置 Master pg_hba.conf，注意這邊和物理複製不一樣，database 必須提供相應的 table：

    ```bash
    # TYPE  DATABASE        USER            ADDRESS                 METHOD
    host    test             logic_repl      172.17.0.3/16           trust
    ```

4. 在 Master 上建立 PUBLICATION：

    ```sql
    [postgres @ test] [local]:5432] 07:57:41 > \dRp
                                  List of publications
     Name | Owner | All tables | Inserts | Updates | Deletes | Truncates | Via root
    ------+-------+------------+---------+---------+---------+-----------+----------
    (0 rows)
    
    [postgres @ test] [local]:5432] 07:57:59 > CREATE PUBLICATION repltest FOR TABLE test_schema.test_table;
    
    [postgres @ test] [local]:5432] 08:00:43 > \dRp
                                    List of publications
      Name  |  Owner   | All tables | Inserts | Updates | Deletes | Truncates | Via root
    --------+----------+------------+---------+---------+---------+-----------+----------
     mytest | postgres | f          | t       | t       | t       | t         | f
    (1 row)
    ```

5. 在 Slave 上建立 SUBSCRIPTION

    ```sql
    [postgres @ test] [local]:5432] 08:00:22 > \dRs
            List of subscriptions
     Name | Owner | Enabled | Publication
    ------+-------+---------+-------------
    (0 rows)
    
    [postgres @ test] [local]:5432] 08:02:30 > CREATE SUBSCRIPTION mysub CONNECTION 'dbname=test host=172.17.0.2 user=logic_repl'  PUBLICATION repltest WITH (copy_data = false);
    
    [postgres @ test] [local]:5432] 08:45:07 > \dRs
              List of subscriptions
     Name  |  Owner   | Enabled | Publication
    -------+----------+---------+-------------
     mysub | postgres | t       | {mytest}
    ```


## 參考

[PostgreSQL: Documentation: 15: Chapter 27. High Availability, Load Balancing, and Replication](https://www.postgresql.org/docs/15/high-availability.html)

[PostgreSQL: Documentation: 15: Chapter 31. Logical Replication](https://www.postgresql.org/docs/15/logical-replication.html)

[PostgreSQL: Documentation: 15: 20.6. Replication](https://www.postgresql.org/docs/current/runtime-config-replication.html)

[PgSQL · 特性分析 · PG主备流复制机制 (taobao.org)](http://mysql.taobao.org/monthly/2015/10/04/)

[PgSQL · 内核解析 · 同步流复制实现分析 (taobao.org)](http://mysql.taobao.org/monthly/2018/01/03/)

[PostgreSQL 9.6 同步多副本 与 remote_apply事务同步级别](https://github.com/digoal/blog/blob/master/201610/20161006_02.md)

[一文彻底弄懂PostgreSQL流复制(全网最详细)_pg流复制_foucus、的博客-CSDN博客](https://blog.csdn.net/weixin_39540651/article/details/106122610)

[PgSQL · 特性分析· Replication Slot (taobao.org)](http://mysql.taobao.org/monthly/2015/02/03/)

[postgresql-14流复制部署手册_HistSpeed的博客-CSDN博客](https://blog.csdn.net/gaojiebao/article/details/121900737)

[Setup PostgreSQL 14 Streaming Replication | Girders: the blog of Allen Fair](https://girders.org/postgresql/2021/11/05/setup-postgresql14-replication/)

[我是一个插槽，今天我做掉了数据库 - 知乎 (zhihu.com)](https://zhuanlan.zhihu.com/p/311496301)

[PG复制状态监控_wal_status reserved_三思呐三思的博客-CSDN博客](https://blog.csdn.net/weixin_37692493/article/details/121089674)

[It’s All About Replication Lag in PostgreSQL (percona.com)](https://www.percona.com/blog/replication-lag-in-postgresql/)