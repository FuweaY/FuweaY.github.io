---
title: MySQL8.0 skip scan range 優化
description: 介紹 MySQL8.0 新增的查詢優化 - skip scan range
slug: skip-scan-range
date: 2021-02-26T12:00:00+08:00
categories:
   - MySQL
tags:
   - "8.0"
weight: 1  
---
# Skip Scan Range

當查詢想要透過 `index`  優化時，需要遵循最左前綴的原則，意即若有 `index(A,B)`，查詢條件只有 `A = ?` 和 `A = ? AND B = ?` 才能吃到這個 `index`， `B = ?` 的條件則無法利用到這個 `index`

從 MySQL 8.0.13 開始新增了一個優化，讓某些情況下 B = ? 的條件可以透過這個 index 優化

```sql
CREATE TABLE t1 (id int NOT NULL AUTO_INCREMENT,f1 INT NOT NULL, f2 INT NOT NULL, PRIMARY KEY(id),KEY test(`f1`,`f2`));
INSERT INTO t1(f1,f2) VALUES(1,1), (1,2), (1,3), (1,4), (1,5),(2,1), (2,2), (2,3), (2,4), (2,5);
INSERT INTO t1(f1,f2) SELECT f1, f2 + 5 FROM t1;
INSERT INTO t1(f1,f2) SELECT f1, f2 + 10 FROM t1;
INSERT INTO t1(f1,f2) SELECT f1, f2 + 20 FROM t1;
INSERT INTO t1(f1,f2) SELECT f1, f2 + 40 FROM t1;
ANALYZE TABLE t1;

EXPLAIN SELECT f1, f2 FROM t1 WHERE f2 > 40;
-- 8.0.13以上，並且skip_scan=on (default)
+----+-------------+-------+------------+-------+---------------+------+---------+------+------+----------+----------------------------------------+
| id | select_type | table | partitions | type  | possible_keys | key  | key_len | ref  | rows | filtered | Extra                                  |
+----+-------------+-------+------------+-------+---------------+------+---------+------+------+----------+----------------------------------------+
|  1 | SIMPLE      | t1    | NULL       | range | f1            | f1   | 8       | NULL |   53 |   100.00 | Using where; Using index for skip scan |
+----+-------------+-------+------------+-------+---------------+------+---------+------+------+----------+----------------------------------------+

-- 8.0.13以下，或 skip_scan=off
+----+-------------+-------+------------+-------+---------------+------+---------+------+------+----------+--------------------------+
| id | select_type | table | partitions | type  | possible_keys | key  | key_len | ref  | rows | filtered | Extra                    |
+----+-------------+-------+------------+-------+---------------+------+---------+------+------+----------+--------------------------+
|  1 | SIMPLE      | t1    | NULL       | index | NULL          | f1   | 8       | NULL |  160 |    33.33 | Using where; Using index |
+----+-------------+-------+------------+-------+---------------+------+---------+------+------+----------+--------------------------+
```

實現方式:1. 在索引前綴(f1) scan 出 distinct 值2. 對其餘索引欄位(f2)，建構subrange scan簡單來說，就是會將其轉化為多個子範圍掃描，以此範例有點類似以下查詢：

```
SELECT f1, f2 FROM t1 WHERE f1 = 1 AND f2 > 40;
SELECT f1, f2 FROM t1 WHERE f1 = 2 AND f2 > 40;
```

主要限制：

1. 必須是複合索引，EX：KEY(A,B,C)

2. 只使用了一張表

3. 不能有 group by 和 select distinct

4. 不能回表，意即 query 中的select、where 只有使用該 index(含pk) 的欄位

參考：

1. [MySQL 官方文檔](https://dev.mysql.com/doc/refman/8.0/en/range-optimization.html#range-access-skip-scan)
2. [MySQL WL#11322](https://dev.mysql.com/worklog/task/?spm=a2c4e.11153940.blogcont696936.10.136121c7o2rRhm&id=11322)
3. [數據庫內核月報(2019/05)](http://mysql.taobao.org/monthly/2019/05/06/