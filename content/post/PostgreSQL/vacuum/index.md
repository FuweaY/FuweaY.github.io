---
title: PostgreSQL vacuum
description: 紀錄學習的 PostgreSQL vacuum
slug: Postgre-vacuum
date: 2020-02-28T12:00:00+08:00
categories:
   - PostgreSQL
weight: 1  
---
在 MySQL 中透過 undo log 來實現 MVCC 的機制，但在 PG 中是將新舊數據都存放在表中，由此產生了表膨脹及 vacuum 的機制。

## MVCC 常用實現方式

一般MVCC有2种实现方法：

- 写新数据时，把旧数据转移到一个单独的地方，如回滚段中，其他人读数据时，从回滚段中把旧的数据读出来，如Oracle数据库和MySQL中的innodb引擎。
- 写新数据时，旧数据不删除，而是把新数据插入。PostgreSQL就是使用的这种实现方法。

两种方法各有利弊，相对于第一种来说，PostgreSQL的MVCC实现方式优缺点如下：

- 优点
    - 无论事务进行了多少操作，事务回滚可以立即完成
    - 数据可以进行很多更新，不必像Oracle和MySQL的Innodb引擎那样需要经常保证回滚段不会被用完，也不会像oracle数据库那样经常遇到“ORA-1555”错误的困扰
- 缺点
    - 旧版本的数据需要清理。当然，PostgreSQL 9.x版本中已经增加了自动清理的辅助进程来定期清理
    - 旧版本的数据可能会导致查询需要扫描的数据块增多，从而导致查询变慢

也就是說 PostgresSQL 的 MVCC 實現方式會導致表中堆積無用的舊數據導致表空間的膨脹問題，也因此需要透過定期的 vacuum 或者 auto vacuum 來釋放這些無用的舊數據占用的空間。

## vacuum 指令

支持以下選項

- `FULL [ ***boolean*** ]`：將表的全部內容重寫到新的 Disk 文件，也就是說能夠確實釋放所有 dead tuple 占用的空間給 OS，但是在完成操作之前不會釋放舊副本的Disk 空間，並且在這期間該表會被加上 `ACCESS EXCLUSIVE` LOCK 阻塞所有對該表的操作。

  備註：沒有 FULL 選項的 vacuum 只是回收空間供該表能夠重複使用，並不會將空間釋放給 OS。

  備註：類似於 MySQL OPTIMIZE TABLE。

- `FREEZE [ ***boolean*** ]`：選擇積極的 freezing，等同於在 `vacuum_freeze_min_age`**、**`vacuum_freeze_table_age` 設置為 0 時運行 vacuum。

  備註：選項 `FULL` 總是會積極的 freezing，因此指定 `FULL` 選項時無需附帶 `FREEZE` 選項。

- `VERBOSE [ ***boolean*** ]`：打印 VACUUM 期間的詳細資訊。
- `ANALYZE [ ***boolean*** ]`：執行 cacuum 後執行 ANALYZE，也就是會更新 planner 使用的統計資訊，確保生成高效的執行計畫。類似於 MySQL 的 ANALYZE TABLE。
- `DISABLE_PAGE_SKIPPING [ ***boolean*** ]` ：此選項禁用 page_skpping 行為，通常只有在導致數據庫損壞的軟硬件問題時才需要使用。
- `SKIP_LOCKED [ ***boolean*** ]` ：指定 VACUUM 在開始處理 relation 時不等待其他衝突鎖的釋放直接跳過。
- `INDEX_CLEANUP { AUTO | ON | OFF }` ：默認值為 AUTO，允許 VACUUM 在適當的情況下跳過 index cleanup 階段。
- `PROCESS_TOAST [ ***boolean*** ]` ：此為默認行為，指定 Vacuum 應該嘗試處理對應的 TOAST 表。

  備註：使用選項 `FULL` 時，還需要包含此選項。

- `TRUNCATE [ ***boolean*** ]` ：指定 VACUUM 應該截斷表末位的空頁，並將其 Disk 空間返回給 OS。

  此為默認行為，除非 vacuum_truncate = false。

  備註：包含選項 `FULL` 時，此選項無效。

- `PARALLEL ***integer***`：指定 index vacuum、index cleanup 階段時並行縣程數量。

  此選項還受 `max_parallel_maintenance_workers` 限制上限，並且只有當 INDEX 大小 > `min_parallel_index_scan_size` 時，該 INDEX 才能 parallel vacuum。

  備註：不能與 `FULL` 選項一起指定。


帶有 full 選項的 vacuum 可以在 `pg_stat_progress_cluster` 觀察進度，如果不帶有 full 選項的 vacuum 則是在 `pg_stat_progress_vacuum` 中觀察進度。

## AutoVacuum

### 簡述

PostgreSQL中的MVCC机制同时存储新旧版本的元组，对于经常更新的表来说，会造成表膨胀的情况。为了解决这个问题，PostgreSQL 引入了 [VACUUM](https://www.postgresql.org/docs/current/static/sql-vacuum.html) 和 [ANALYZE](https://www.postgresql.org/docs/10/static/sql-analyze.html) 命令，并且引入了AutoVacuum自动清理。

在PostgreSQL中，AutoVacuum自动清理操作包括：

- 删除或重用无效元组的磁盘空间
- 更新数据统计信息，保证执行计划更优
- 更新visibility map，加速index-only scans
- 避免XID 回卷(wraparound)造成的数据丢失

  在 PG 中 XID 是用32位无符号数来表示的，也就是说如果不引入特殊的处理，当PostgreSQL的XID 到达40亿，会造成溢出，从而新的XID 为0。而按照PostgreSQL的MVCC 机制实现，之前的事务就可以看到这个新事务创建的元组，而新事务不能看到之前事务创建的元组，这违反了事务的可见性。本文将这种现象称为XID 的回卷问题。

  備註：MySQL 為 48 位約 281 兆的量，雖然理論上會發生同樣的狀況，但實務上非常困難。


为了实现自动清理，PostgreSQL引入了两种类型的辅助进程：

- autovacuum launcher：AutoVacuum机制的守护进程，周期性地调度autovacuum worker进程。
- autovacuum worker：进行具体的自动清理工作。

### 相關參數

#### **Automatic Vacuuming**

- autovacuum (boolean)：是否啟用 auto vacuum 功能，需要啟用 track_counts (默認)，默認是開啟的，另外此參數可以 BY Table 單獨設置。
- autovacuum_max_workers (integer)：指定 auto vacuum 進程數 (不包含 launcher) 默認值為 3。
- autovacuum_naptime (integer)：設置兩次 auto vacuum 之間的間隔時間，默認值為 1min。
- autovacuum_vacuum_threshold (integer)：與 `autovacuum_vacuum_scale_factor` 搭配使用，默認值為 50。
- autovacuum_vacuum_scale_factor (floating point)：當一張表的 update 或 delete 的元組 (tuples) 數超過 `autovacuum_vacuum_threshold` + `autovacuum_vacuum_scale_factor` * `table size` 會觸發 vacuum。預設值為 0.2 (表大小的 20%)，另外此參數可以 BY Table 單獨設置。
- autovacuum_analyze_threshold (integer)：與 `autovacuum_analyze_scale_factor` 搭配使用，默認值為 50。
- autovacuum_analyze_scale_factor (floating point)：當一張表的 update 或 delete 的元組 (tuples) 數超過 `autovacuum_analyze_threshold` + `autovacuum_analyze_scale_factor` * `table size` 會觸發 ANALYZE。預設值為 0.1 (表大小的 10%)，另外此參數可以 BY Table 單獨設置。
- autovacuum_vacuum_insert_threshold (integer)：與 `autovacuum_vacuum_insert_scale_factor` 搭配使用，默認值為 1000。
- autovacuum_vacuum_insert_scale_factor (floating point)：當一張表 insert 指定的元組 (tuples) 數達到 `autovacuum_vacuum_insert_threshold` + `autovacuum_vacuum_insert_scale_factor` * `table size` 會觸發 vacuum。預設值為 0.2 (表大小的 20%)，另外此參數可以 BY Table 單獨設置。
- autovacuum_freeze_max_age (integer)：設置當 XID (`pg_class`
  .`relminmxid`)達到此上限值時，必須強制 vacuum 避免 XID 的 wraparound。默認值為 2 億。

  注意：即使 autovacuum 沒有開啟仍舊會觸發強制 vacuum 。

- autovacuum_multixact_freeze_max_age (integer)：設置當 multi XID (`pg_class`
  .`relminmxid`)上限值達到此上限值時，必須強制 vacuum 避免 XID 的 wraparound。默認值為 4 億。

  注意：即使 autovacuum 沒有開啟仍舊會觸發強制 vacuum 。

- autovacuum_vacuum_cost_limit (integer)：設置 auto vacuum 的開銷限制，默認值為 -1 表示使用 `vacuum_cost_limit` 的設定。開銷的計算依據 `vacuum_cost_page_hit`, `vacuum_cost_page_miss`, `vacuum_cost_page_dirty` 的設置。
- autovacuum_vacuum_cost_delay (floating point)：設置如過超過 `autovacuum_vacuum_cost_limit` 上的開銷限制，則需要延遲多少時間清理，默認值為 2 (ms)，當此設定設置為 -1 時表示依照 `vacuum_cost_delay` 的設定。

#### RESOURCE USAGE

- autovacuum_work_mem (integer)：指定每個 autovacuum worker 進程使用的最大 memory 量，默認值為 -1 表示使用 `maintenance_work_mem`。

  建議單獨設置，因為實際最多可能會消耗  `autovacuum_max_workers` * `autovacuum_work_mem` 量的 memory，這部分和 `maintenance_work_mem` * `max_connections` 不同，因此為了更加安全的限制資源使用量應該分開設置。

- vacuum_cost_page_hit (integer)：清理一个在共享缓存中找到的缓冲区的估计代价。它表示锁住缓冲池、查找共享哈希表和扫描页内容的代价。默认值为1。
- vacuum_cost_page_miss (integer)：清理一个必须从磁盘上读取的缓冲区的代价。它表示锁住缓冲池、查找共享哈希表、从磁盘读取需要的块以及扫描其内容的代价。默认值为10。
- vacuum_cost_page_dirty (integer)：当清理修改一个之前干净的块时需要花费的估计代价。它表示再次把脏块刷出到磁盘所需要的额外I/O。默认值为20。
- vacuum_cost_limit (integer)：設置 vacuum 的開銷限制達到此設置時休眠 `vacuum_cost_delay` 的設置，默認值為 200。開銷的計算依據 `vacuum_cost_page_hit`, `vacuum_cost_page_miss`, `vacuum_cost_page_dirty` 的設置。
- vacuum_cost_delay (floating point)：如過超過 vacuum 開銷成本達到 `vacuum_cost_limit` 限制時需要休眠多久，默認值為 0 表示禁用基於成本計算的 delay 功能。> 0 表示開啟，但一般也不建議設置超過 > 1ms，建議可以保持默認值關閉就好。

#### 其他

- track_counts (boolean)：是否啟用統計信息收集功能，默認是開啟的，因為 autovacuum 的 launcher 進程需要這些統計資訊。
- log_autovacuum_min_duration (integer)：當 auto vacuum 運行超過該時間或因鎖衝突而退出則會記錄到 log 中，預設值為 10 (min)，若設為 -1 表示禁用。

## 設置建議

固定

```bash
vacuum_cost_delay = 0 
# 以下無用                     
vacuum_cost_page_hit = 1                     
vacuum_cost_page_miss = 10                 
vacuum_cost_page_dirty = 20                
vacuum_cost_limit = 10000  
# 以上無用             

autovacuum = on  
log_autovacuum_min_duration = 0  
autovacuum_analyze_scale_factor = 0.05  
autovacuum_freeze_max_age = 1200000000  
autovacuum_multixact_freeze_max_age = 1400000000  
autovacuum_vacuum_cost_delay=0
autovacuum_vacuum_scale_factor = 0.02     # 0.005~ 0.15

vacuum_freeze_table_age = 200000000  
vacuum_multixact_freeze_table_age = 200000000
```

變動

```bash
autovacuum_work_mem           # min( 8G, (规格内存*1/8)/autovacuum_max_workers )
autovacuum_max_workers        # max(min( 8 , CPU核数/2 ) , 5)
```

## 後記

在 PostgreSQL 中因為透過保存舊的 tuples (元組) 來實現 MVCC 機制，容易造成數據的膨脹進而導致儲存空間的消耗也可能降低查詢的效能。

在開源社區也有持續進行討論，目前內部比較認可的方式是構建一種新的儲存格式，也就是由 *EnterpriseDB 公司主導的 zheap。

*EnterpriseDB 公司提供基於 PostgreSQL 的企業級產品與服務廠商。

## 參考

[PostgreSQL: Documentation: 15: VACUUM](https://www.postgresql.org/docs/current/sql-vacuum.html)

[PostgreSQL: Documentation: 15: 25.1. Routine Vacuuming](https://www.postgresql.org/docs/current/routine-vacuuming.html)

[PgSQL · 特性分析 · MVCC机制浅析 (taobao.org)](http://mysql.taobao.org/monthly/2017/10/01/)

[PgSQL · 答疑解惑 · 表膨胀 (taobao.org)](http://mysql.taobao.org/monthly/2015/12/07/)

[PgSQL · 源码分析 · AutoVacuum机制之autovacuum launcher (taobao.org)](http://mysql.taobao.org/monthly/2017/12/04/)

[PgSQL · 源码分析 · AutoVacuum机制之autovacuum worker (taobao.org)](http://mysql.taobao.org/monthly/2018/02/04/)

[PgSQL · 新特性解读 · undo log 存储接口（上） (taobao.org)](http://mysql.taobao.org/monthly/2019/07/02/)

[PgSQL · 特性分析 · 事务ID回卷问题 (taobao.org)](http://mysql.taobao.org/monthly/2018/03/08/)

[blog/20181203_01.md at master · digoal/blog · GitHub](https://github.com/digoal/blog/blob/master/201812/20181203_01.md)

[【Postgresql】VACUUM 垃圾回收 - 知乎 (zhihu.com)](https://zhuanlan.zhihu.com/p/585906727)

[PostgreSQL VACUUM 之深入浅出 (一) - DBADaily - 博客园 (cnblogs.com)](https://www.cnblogs.com/dbadaily/p/vacuum1.html)

[PostgreSQL 事务—MVCC_postgresql mvcc_obvious__的博客-CSDN博客](https://blog.csdn.net/obvious__/article/details/120710977?spm=1001.2014.3001.5502)

[PostgreSQL Vacuum---元组删除_heap_page_obvious__的博客-CSDN博客](https://blog.csdn.net/obvious__/article/details/121318928?spm=1001.2014.3001.5502)