---
title: PostgreSQL 備份還原
description: 紀錄學習的 PostgreSQL 備份還原
slug: Postgre-backup-restore
date: 2020-02-26T12:00:00+08:00
categories:
   - PostgreSQL
weight: 1  
---
## pg_dump

就像 mysqldump 一樣的效果，將數據 dump 成 SQL 語句。

### 備份

使用 pg_dump 來備份單個 database：

```bash
/usr/bin/pg_dump test > test.dump
```

```bash
➜  ~ cat test.dump
...
CREATE TABLE public.test (
    id integer
);

ALTER TABLE public.test OWNER TO postgres;
COPY public.test (id) FROM stdin;
1
2
3
4
\.
...
```

預設情況下 dump 出來的備份檔是以 COPY 語句來還原數據，如果需要將 copy 語法變為 insert 可以使用 `column-inserts` 並透過 `rows-per-insert` 指定一個 insert 語句包含的行數：

```bash
/usr/bin/pg_dump --column-inserts --rows-per-insert 2 test > test.dump

# test.dump 中的部分內容
INSERT INTO public.test (id) VALUES
        (1),
        (2);
INSERT INTO public.test (id) VALUES
        (3),
        (4);
```

### 還原

使用 psql 來還原：

```bash
# 建立要還原的目標 database
psql --command "create database test_restore"
# 還原
psql test_restore < test.dump
```

預設情況下還原遇到錯誤繼續執行，如果希望可以發生 SQL 錯誤時退出，可以使用以下方式：

```bash
psql --set ON_ERROR_STOP=on dbname < dumpfile
```

如果還需要更進一步將整個還原過程視為一個 Transaction，可以透過添加 --single-transaction：

```bash
psql --set ON_ERROR_STOP=on --single-transaction dbname < dumpfile
```

此外還可以透過 pipeline 的方式在 pg_dump 的同時直接還原：

```bash
pg_dump -h host1 dbname | psql -h host2 dbname
```

### 其他事項

1. pg_dump 的 User 需要對該 Table 有讀取權限。
2. pg_dump 一次只能備份一個 database，並且不包含 role、tablespace (因為他們屬於 cluster 範圍，而不是 database)。
3. pg_dump 運行時會執行 database snapshot 來達到內部一致性。
4. pg_dump 會和 ALTER 阻塞。
5. 要一次備份整個 cluster 請使用 pg_dumpall，這會 dump 所有的 database 和 cluster 範圍的數據，例如 role、tablespace。

    ```bash
    # 備份
    pg_dumpall > dumpfile
    # 還原
    psql -f dumpfile postgres
    ```

   但是需要注意因為包含 cluster 範圍的數據，因此必須使用 super user 的權限。

   注意：pg_dumpall 實際上會為每個 database 調用 pg_dump，也就是不保證所有 database snapshot 是同步的。

   此外可以透過 `--globals-only` 選項來只有備份 cluster 範圍的數據。

6. 如果在 pg_dump 時使用非純文本的格式(-Fc) 時，可以使用 pg_restore 來還原。

優點：

1. 可以還原到不同的 RDBMS。
2. 可以還原到不同版本的 PostgreSQL。

缺點：

1. 因為等於需要運行每一個 SQL 命令來還原，因此還原速度較慢。

## pg_basebackup

為 PostgreSQL 內建的物理備份工具，用於對 PostgreSQL 進行全量物理熱備份。

pg_basebackup 會在備份期間創建一個備份歷史文件，該文件會以備份時的第一個 WAL 文件為命名，例如 `000000010000000000000002.00000028.backup` 表示備份時第一個 WAL 名稱是 `000000010000000000000002` 另外的 `00000028` 表示其中的確切位置 (一般來說可以忽略)，以下是該檔案包含的資訊：

```sql
[root@ed875d053382 pg_wal]# cat 000000010000000000000002.00000028.backup
START WAL LOCATION: 0/2000028 (file 000000010000000000000002)
STOP WAL LOCATION: 0/2000100 (file 000000010000000000000002)
CHECKPOINT LOCATION: 0/2000060
BACKUP METHOD: streamed
BACKUP FROM: primary
START TIME: 2023-04-17 07:07:01 UTC
LABEL: pg_basebackup base backup
START TIMELINE: 1
STOP TIME: 2023-04-17 07:07:02 UTC
STOP TIMELINE: 1
```

也就是說只需要保存包含 `000000010000000000000002` 之後的 WAL 就可以完整的復原。

### 範例

```bash
pg_basebackup -P -D /var/data/backup/pg_back_$(date +"%F_%T")
```

### 選項

- -D：指定備份的目的地
- -F、：指定輸出的格式
    - p、plain：預設值，輸出成普通文件。
    - t、tar：輸出為 tar 文件。
- -R：創建 standy.signal 文件，並將相關的連接設置附加到 `postgresql.auto.conf` 中，用於便於將備份檔用於建置 replica。
- -X：指定備份在備份期間中產生的 WAL。
    - n、none：不備份 WAL。
    - f、fetch：將在備份結束後收集 WAL，因此需要有足夠高的 `wal_keep_size`，否則可能發生所需的 wal 被回收導致備份無法使用。
    - s、stream：默認值，在備份的同時透過 streaming replication 持續傳輸 WAL，這將額外開一個連結來進行 WAL 的備份，
- -P：顯示進度。

### 其他事項

優點：

缺點：只能備份整個 Instance 無法指定單表。

## Continuous archiving 與 PITR

只要是數據庫都有最基本的 Durability(持久性) 需要保證，這大部分都是透過 WAL 技術也就是日誌先行來達到，在 MySQL 中是 REDO LOG 而在 PostgreSQL 中則是 WAL (在 PG 10 之前稱為 xlog)。

因此就像 MySQL 可以透過 xtrabackup 的物理備份加上 binlog 做後續的增量恢復，PostgreSQL 同樣可以透過 pg_basebackup 產生的全量備份加上後續的 WAL 達到增量恢復，這樣還有 *point-in-time recovery* 的優勢。

### 設置 WAL Archining

PostgreSQL 每 16 MB (預設值)就會輪換一個新的 WAL 文件，而當所有的 WAL 超過 max_wal_size 舊的 WAL 會被一一刪除 (依照 checkpoint 判斷是否不再需要對應的 WAL)，為了避免全量備份後所需的 WAL 在經過一陣子後被刪除導致整個全量備份不可用，我們需要為 WAL 設置 Archining 的方式，確保在刪除 WAL 之前有進行歸檔。

要配置 WAL Archining 需要以下設定：

- wal_level = replica | logical
- archive_mode = ON
- arvhive_command = ‘test ! -f /mnt/server/archivedir/%f && cp %p /mnt/server/archivedir/%f’
    - %f  會被替換成歸檔的 wal 文件名稱。
    - %p 會被替換成歸檔的 wal 路徑。

```bash
wal_level = replica
archive_mode = ON
arvhive_command = 'test ! -f /var/data/xlog_archive/%f && cp %p /var/data/xlog_archive/%f' 
```

### 範例

1. 透過 pg_basebackup 獲取全量備份

    ```bash
    pg_basebackup -P -D /var/data/backup/pg_back_$(date +"%F_%T")
    ```

2. 關閉 server：

    ```bash
    pg_ctl stop
    ```

3. 對當前的實體檔案暫存，以備不時之需：

    ```bash
    cp -R /var/data/postgres/ /var/data/postgres_backup/ 
    ```

   備註：如果沒有足夠的空間應至少保留 pg_wal 子目錄，因為可能包含尚未 archive 的 wal 需用於稍後恢復。

4. 清除當前的實體檔案

    ```bash
    rm -rf /var/data/postgres/*
    ```

5. 從 Base Backup 進行恢復，注意請使用 DB系統用戶 (postgre) 進行包含正確權限的恢復。

    ```bash
    cp -R /var/data/backup/pg_back_2023-04-13_02\:55\:49/* /var/data/postgres/
    ```

   如果有使用 tablespace 應該檢查 pg_tblspc 中的 symbolic link 是否正確恢復。

6. 移除 Base Backup 中的 `pg_wal/` 下的所有文件，因為這些是當初 Base Backup 的 wal 文件可能已經過時了

    ```bash
    rm -rf /var/data/postgres/pg_wal/*
    ```

7. 將步驟 2 中保存的未歸檔 wal 文件複製到 pg_wal

    ```bash
    cp -R /var/data/postgres_backup/pg_wal/* /var/data/postgres/pg_wal/
    ```

8. 在 `postgresql.conf` 中設置恢復相關的配置

    ```bash
    restore_command = 'cp /var/data/xlog_archive/%f %p'
    # recovery_target_time 可以配置恢復到的時間點
    # recovery_target_timeline 
    ```

   建立 `recovery.signal` 文件來觸發 restore

    ```bash
    touch recovery.signal
    ```

   注意：此時還可以調整 pg_hba.conf 來防止其他 role 來連線剛恢復的 cluster。

9. 啟動 server

    ```bash
    pg_ctl -D /var/data/postgres -l logfile start
    ```

   恢復成功後 `recovery.signal` 會被刪除，避免稍後意外重新進入恢復模式。

10. 檢查數據庫是否恢復完成，如果完成記得重新調整 pg_hba.conf 恢復其他使用者的存取。

### 實例

1. 新增一些測試資料：

    ```sql
    create database test;
    \c test;
    create table test_table(id int primary key,name varchar(10));
    
    [postgres @ test] [local]:5432] 02:53:21 > select * from test_table;
     id | name
    ----+-------
      1 | test
      2 | test2
    (2 rows)
    ```

2. 進行 Base backup：

    ```bash
    pg_basebackup -P -D /var/data/backup/pg_back_$(date +"%F_%T")
    ```

3. 新增增量測試資料：

    ```sql
    create table test_table2(id int primary key,name varchar(10));
    insert into test_table values(3,'test3'),(4,'test4');
    insert into test_table2 values(5,'test5'),(6,'test6');
    
    [postgres @ test] [local]:5432] 02:57:25 > select * from test_table;
     id | name
    ----+-------
      1 | test
      2 | test2
      3 | test3
      4 | test4
    (4 rows)
    
    Time: 0.381 ms
    [postgres @ test] [local]:5432] 02:57:29 > select * from test_table2;
     id | name
    ----+-------
      5 | test5
      6 | test6
    (2 rows)
    ```

4. 強制關閉 pg 服務

    ```bash
    [postgres@3f5b5718e08f postgres]$ ps aux| grep postgre
    root        801  0.0  0.1  83584  2196 pts/0    S    02:14   0:00 su postgres
    postgres    802  0.0  0.1  14096  2888 pts/0    S    02:14   0:00 bash
    postgres   1148  0.0  1.0 216872 19112 ?        Ss   02:56   0:00 /usr/local/postgres/bin/postgres
    postgres   1150  0.0  0.0 216872  1568 ?        Ss   02:56   0:00 postgres: checkpointer
    postgres   1151  0.0  0.1 217008  2596 ?        Ss   02:56   0:00 postgres: background writer
    postgres   1152  0.0  0.5 216872  9720 ?        Ss   02:56   0:00 postgres: walwriter
    postgres   1153  0.0  0.1 217428  2776 ?        Ss   02:56   0:00 postgres: autovacuum launcher
    postgres   1154  0.0  0.0 216872  1580 ?        Ss   02:56   0:00 postgres: archiver
    postgres   1155  0.0  0.0  67512  1756 ?        Ss   02:56   0:00 postgres: stats collector
    postgres   1156  0.0  0.1 217428  2316 ?        Ss   02:56   0:00 postgres: logical replication launcher
    postgres   1173  0.0  0.1  53332  1876 pts/0    R+   02:57   0:00 ps aux
    postgres   1174  0.0  0.0  10696   984 pts/0    S+   02:57   0:00 grep --color=auto postgre
    [postgres@3f5b5718e08f postgres]$ kill -9 postgres
    ```

5. 備份當前實體檔案並清除

    ```bash
    cp -R /var/data/postgres/ /var/data/postgres_backup/ 
    rm -rf /var/data/postgres/*
    ```

6. 從 Base Backup 進行恢復，並清空備份中的 wal，再將原本的 wal 檔案複製過來

    ```bash
    cp -R /var/data/backup/pg_back_2023-04-13_02\:55\:49/* /var/data/postgres/
    rm -rf /var/data/postgres/pg_wal/*
    cp -R /var/data/postgres_backup/pg_wal/* /var/data/postgres/pg_wal/
    ```

7. 配置 restore_command

    ```bash
    restore_command = 'cp /var/data/xlog_archive/%f %p'
    ```

   建立 `recovery.signal` 文件來觸發 restore

    ```bash
    touch recovery.signal
    ```

8. 啟動並檢查

    ```bash
    [postgres@3f5b5718e08f postgres]$ pg_ctl -D /var/data/postgres -l logfile start
    
    [postgres@3f5b5718e08f postgres]$ psql test
    Timing is on.
    psql (14.7)
    Type "help" for help.
    
    [postgres @ test] [local]:5432] 03:00:30 > \dt
                List of relations
     Schema |    Name     | Type  |  Owner
    --------+-------------+-------+----------
     public | test_table  | table | postgres
     public | test_table2 | table | postgres
    (2 rows)
    
    [postgres @ test] [local]:5432] 03:00:32 > select * from test_table;
     id | name
    ----+-------
      1 | test
      2 | test2
      3 | test3
      4 | test4
    (4 rows)
    
    Time: 0.620 ms
    [postgres @ test] [local]:5432] 03:00:37 > select * from test_table2;
     id | name
    ----+-------
      5 | test5
      6 | test6
    (2 rows)
    
    Time: 0.715 ms
    ```


### Point-in-time recovery

預設情況下會直接恢復到 WAL 的末尾，可以透過設置 recovery_target 相關參數來指定一個更早的停止點，達到 Point-in-time recovery。

相關參數如下：

- recovery_target = ’immediate’：目前只有 `immediate` 此設定，表示盡早結束也就是該備份結束的時間點。
- recovery_target_name (string)：指定透過 `pg_create_restore_point()` 所創建的 restore point，指示恢復到此位置。
- recovery_target_time (timestamp)：指定恢復到指定的時間戳。

  準確來說是指 WAL 中記錄

- recovery_target_xid (string)：指定恢復到指定的 Tranasction ID，注意雖然 Transaction ID 是順序分配的，但是可能依照不同順序 commit，此設置同時也會恢復該 Transaction ID 之前 commit 但具有更大的 Transaction ID 的事務。
- recovery_target_lsn (string)：指定恢復到指定的 LSN 位置。
- recovery_target_inclusive (boolean)：影響當設置 `recovery_target_time`、 `recovery_target_xid` 和 `recovery_target_lsn`，默認值 ON 表示恢復完這些目標才停止，如果是 OFF 則表示恢復到這些設置之前就停止。
- recovery_target_timeline (string)：指定恢復到一個 timeline 中，該值可以 timeline id 或以下特定值：
    - current：沿著 base backup 的 timeline 進行恢復。
    - latest：默認值，恢復到最新的 timeline。
- recovery_target_action (enum)：指定當達到 recovery target 後的動作：
    - pause：默認值，表示停止 recovery 並允許查詢。

      如果 hot_standby = off，則此值行為和 shutdown 等同。

      如果在 promote 中達到 recovery target，則此值行為和 promote 等同。

    - promote：表示恢復結束後 sever 將允許連接。
    - shutdown：表示恢復結束後停止 server，設置此值時 recovery.singal 將不會被移除。

  備註：如果沒有設置 recovery target 相關參數，則此參數無效。



> 💡 注意：recovery_target、recovery_target_name、recovery_target_time、recovery_target_xid 、recovery_target_lsn 只能夠設置其中一個，設置多個將引發錯誤。

- 實做
    1. 當前資料庫的資料內容

        ```sql
        [postgres @ test] [local]:5432] 03:24:20 > select * from test;
         id | timeline |        create_time
        ----+----------+----------------------------
          1 |        1 | 2023-04-19 03:07:52.695513
          2 |        1 | 2023-04-19 03:08:06.031361
          3 |        1 | 2023-04-19 03:08:14.686946
          4 |        1 | 2023-04-19 03:15:46.191272
          5 |        1 | 2023-04-19 03:20:29.690359
          6 |        1 | 2023-04-19 03:24:08.854501
        (6 rows)
        ```

    2. 使用 base backup 復原後調整設定檔

        ```sql
        [postgres@31650d651e1c data]$ cp -r backup/pg_back_2023-04-19_03\:08\:41/* postgres/
        [postgres@31650d651e1c postgres]$ vim postgresql.conf
        recovery_target_time = '2023-04-19 03:20:00'
        
        [postgres@31650d651e1c postgres]$ touch recovery.signal
        ```

       備註：可以透過 pg_waldump 確認 wal log 來確認要恢復到的位置，例如我希望恢復到 id = 5 被 insert 之前的狀況，透過 waldump 確認該 transaction 在 `2023-04-19 03:20:29.690738 UTC` commit，因此只要將 recovery_target_time 設置的比其小即可。

        ```sql
        [postgres@31650d651e1c xlog_archive]$ pg_waldump 00000001000000000000000D
        rmgr: Standby     len (rec/tot):     50/    50, tx:          0, lsn: 0/0D000028, prev 0/0C000358, desc: RUNNING_XACTS nextXid 748 latestCompletedXid 747 oldestRunningXid 748
        rmgr: Standby     len (rec/tot):     50/    50, tx:          0, lsn: 0/0D000060, prev 0/0D000028, desc: RUNNING_XACTS nextXid 748 latestCompletedXid 747 oldestRunningXid 748
        rmgr: XLOG        len (rec/tot):    114/   114, tx:          0, lsn: 0/0D000098, prev 0/0D000060, desc: CHECKPOINT_ONLINE redo 0/D000060; tli 1; prev tli 1; fpw true; xid 0:748; oid 24582; multi 1; offset 0; oldest xid 726 in DB 1; oldest multi 1 in DB 1; oldest/newest commit timestamp xid: 0/0; oldest running xid 748; online
        rmgr: Standby     len (rec/tot):     50/    50, tx:          0, lsn: 0/0D000110, prev 0/0D000098, desc: RUNNING_XACTS nextXid 748 latestCompletedXid 747 oldestRunningXid 748
        rmgr: Heap        len (rec/tot):     54/   298, tx:        748, lsn: 0/0D000148, prev 0/0D000110, desc: INSERT off 5 flags 0x00, blkref #0: rel 1663/16384/16391 blk 0 FPW
        rmgr: Btree       len (rec/tot):     53/   193, tx:        748, lsn: 0/0D000278, prev 0/0D000148, desc: INSERT_LEAF off 5, blkref #0: rel 1663/16384/16394 blk 1 FPW
        rmgr: Transaction len (rec/tot):     34/    34, tx:        748, lsn: 0/0D000340, prev 0/0D000278, desc: COMMIT 2023-04-19 03:20:29.690738 UTC
        ```

    3. 啟動 server

        ```sql
        [postgres@945f297dce73 postgres]$ pg_ctl start
        2023-04-19 07:44:25.775 UTC [1828] LOG:  starting point-in-time recovery to 2023-04-19 03:20:00+00
        2023-04-19 07:44:25.817 UTC [1828] LOG:  restored log file "00000001000000000000000B" from archive
        2023-04-19 07:44:25.975 UTC [1828] LOG:  redo starts at 0/B000028
        2023-04-19 07:44:25.977 UTC [1828] LOG:  consistent recovery state reached at 0/B000100
        2023-04-19 07:44:25.977 UTC [1827] LOG:  database system is ready to accept read-only connections
        2023-04-19 07:44:26.008 UTC [1828] LOG:  restored log file "00000001000000000000000C" from archive
        2023-04-19 07:44:26.198 UTC [1828] LOG:  restored log file "00000001000000000000000D" from archive
        2023-04-19 07:44:26.351 UTC [1828] LOG:  recovery stopping before commit of transaction 748, time 2023-04-19 03:20:29.690738+00
        2023-04-19 07:44:26.351 UTC [1828] LOG:  pausing at the end of recovery
        2023-04-19 07:44:26.351 UTC [1828] HINT:  Execute pg_wal_replay_resume() to promote.
        
        [postgres@31650d651e1c postgres]$ psql test
        Timing is on.
        psql (14.7)
        Type "help" for help.
        
        [postgres @ test] [local]:5432] 05:13:41 > select * from test;
        test-# ;
         id | timeline |        create_time
        ----+----------+----------------------------
          1 |        1 | 2023-04-19 03:07:52.695513
          2 |        1 | 2023-04-19 03:08:06.031361
          3 |        1 | 2023-04-19 03:08:14.686946
          4 |        1 | 2023-04-19 03:15:46.191272
        ```



## timeline

想像時間旅行的科幻作品，如果從現在回到過去並改變過去發生的事件會導致未來發生改變，也就是出現了不同的時間線。

在 PG 中也有類似的狀況，假設在 12:00 誤刪除了資料表，並在 13:00 將 PG 恢復到 11:00 的時間點並正常啟動和運作，此時會出現 2 個時間線：

1. 過去 11:00~13:00 產生的 WAL
2. 恢復後從 13:00 開始產生的 WAL

如果沒有 timeline 的設計，恢復後產生的 WAL 覆蓋了過去產生的 WAL，此時如果發現決策錯誤希望回到過去的 12:00 數據庫狀態將無法做到。

因此 PG 有了 timeline 的設計，透過保存不同 timeline 的 WAL 來讓使用者能夠自由的恢復到不同的時間線。

### 新的 timeline 產生時機

1. PITR：設置 recovery_target 相關參數進行恢復後會出現新的 timeline。

    ```sql
    [postgres@31650d651e1c postgres]$ ll pg_wal/
    total 32768
    -rw-------. 1 postgres postgres 16777216 Apr 19 03:27 00000001000000000000000B
    -rw-------. 1 postgres postgres 16777216 Apr 19 03:27 00000001000000000000000C
    drwx------. 2 postgres postgres       80 Apr 19 03:27 archive_status
    
    # 配置 recovery_target 參數並進行 PITR
    [postgres@31650d651e1c postgres]$ vim postgresql.conf
    recovery_target_time = '2023-04-19 03:15:46'
    recovery_target_action = 'promote'
    [postgres@31650d651e1c postgres]$ touch recovery.singal
    [postgres@31650d651e1c postgres]$ pg_ctl -l logfile restart
    waiting for server to shut down.... done
    server stopped
    waiting for server to start.... done
    server started
    
    [postgres@31650d651e1c pg_wal]$ ll
    total 49156
    -rw-------. 1 postgres postgres 16777216 Apr 19 03:30 00000001000000000000000C
    -rw-------. 1 postgres postgres 16777216 Apr 19 03:30 00000002000000000000000C
    -rw-------. 1 postgres postgres 16777216 Apr 19 03:27 00000002000000000000000D
    -rw-------. 1 postgres postgres       50 Apr 19 03:30 00000002.history
    drwx------. 2 postgres postgres       73 Apr 19 03:30 archive_status
    
    [postgres@31650d651e1c pg_wal]$ cat 00000002.history
    1       0/C0002F8       before 2023-04-19 03:15:46.192096+00
    ```

2. standby promote：當 standby 被 promote 成 primary 時會出現新的 timeline。

    ```sql
    [postgres@31650d651e1c pg_wal]$ ll
    total 32768
    -rw-------. 1 postgres postgres 16777216 Apr 19 02:00 000000010000000000000006
    -rw-------. 1 postgres postgres 16777216 Apr 19 02:00 000000010000000000000007
    drwx------. 2 postgres postgres        6 Apr 19 02:00 archive_status
    
    [postgres@31650d651e1c pg_wal]$ pg_ctl promote
    waiting for server to promote.... done
    server promoted
    
    [postgres@31650d651e1c pg_wal]$ ll
    total 49160
    -rw-------. 1 postgres postgres 16777216 Apr 19 02:03 000000010000000000000006.partial
    -rw-------. 1 postgres postgres 16777216 Apr 19 02:03 000000020000000000000006
    -rw-------. 1 postgres postgres 16777216 Apr 19 02:00 000000020000000000000007
    -rw-------. 1 postgres postgres       41 Apr 19 02:03 00000002.history
    drwx------. 2 postgres postgres       80 Apr 19 02:03 archive_status
    
    [postgres@31650d651e1c pg_wal]$ cat 00000002.history
    1	  0/A000198	  no recovery target specified
    ```


### timeline 實體檔案

從上一節中可以看到產生新的 timeline 時會出現後綴相同的 wal 及 history 檔案：

```sql
[postgres@31650d651e1c postgres]$ ll pg_wal/
total 32768
-rw-------. 1 postgres postgres 16777216 Apr 19 03:27 00000001000000000000000B
-rw-------. 1 postgres postgres 16777216 Apr 19 03:27 00000001000000000000000C
drwx------. 2 postgres postgres       80 Apr 19 03:27 archive_status

# 配置 recovery_target 參數並進行 PITR
[postgres@31650d651e1c postgres]$ vim postgresql.conf
recovery_target_time = '2023-04-19 03:15:46'
recovery_target_action = 'promote'
[postgres@31650d651e1c postgres]$ touch recovery.singal
[postgres@31650d651e1c postgres]$ pg_ctl -l logfile restart
waiting for server to shut down.... done
server stopped
waiting for server to start.... done
server started

[postgres@31650d651e1c pg_wal]$ ll
total 49156
-rw-------. 1 postgres postgres 16777216 Apr 19 03:30 00000001000000000000000C
-rw-------. 1 postgres postgres 16777216 Apr 19 03:30 00000002000000000000000C
-rw-------. 1 postgres postgres 16777216 Apr 19 03:27 00000002000000000000000D
-rw-------. 1 postgres postgres       50 Apr 19 03:30 00000002.history
drwx------. 2 postgres postgres       73 Apr 19 03:30 archive_status

[postgres@31650d651e1c pg_wal]$ cat 00000002.history
1       0/C0002F8       before 2023-04-19 03:15:46.192096+00
```

可以觀察到有一個 `00000002.history` 的 timeline 相關檔案，這個件紀錄了這個 timeline 從哪條 timeline 分支出來及什麼時候，這包含了以下 3 個字段：

- parentTLI：parent timeline 的 ID。
- LSN：發生 timline 切換的 WAL 位置。
- reason：timeline 產生的原因。

例如：

```sql
[postgres@31650d651e1c pg_wal]$ cat 00000002.history
1	  0/A000198	before 2023-4-19 12:05:00.861324+00
```

以上表示了 timeline 2 是基於 timeline 1 的 basebackup 透過 WAL 恢復到 0/A000198 對應 `2023-4-19 12:05:00.861324+00` 之前的位置。

另外還可以注意到出現新的 timeline 的時候 WAL 檔名也有了變化：`00000001000000000000000C` → `00000002000000000000000C`，可以觀察到前 8 碼由 `00000001` 變成 `00000002`，事實上 WAL 檔名的前 8 碼就是用來標示其所屬的 timeline id。

# 參考

[PostgreSQL: Documentation: 15: Chapter 26. Backup and Restore](https://www.postgresql.org/docs/current/backup.html)

[PostgreSQL备份恢复实现 - 知乎 (zhihu.com)](https://zhuanlan.zhihu.com/p/410766742)

[1.pg_basebackup 介绍及使用 - www.cqdba.cn - 博客园 (cnblogs.com)](https://www.cnblogs.com/cqdba/p/15920508.html)

[PostgreSQL 從入門到出門 第 8 篇 備份與恢復 - 台部落 (twblogs.net)](https://www.twblogs.net/a/5c9e6330bd9eee73ef4b6004)

[PostgreSQL时间线(timeline)和History File_pg history_foucus、的博客-CSDN博客](https://blog.csdn.net/weixin_39540651/article/details/111239341)

[The Internals of PostgreSQL : Chapter 10 Base Backup & Point-in-Time Recovery (interdb.jp)](https://www.interdb.jp/pg/pgsql10.html#_10.3.1.)