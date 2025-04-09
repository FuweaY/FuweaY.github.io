---
title: MySQL8.0 EXISTS 及 NOT IN/EXISTS 優化
description: 介紹 MySQL8.0 EXISTS 及 NOT IN/EXISTS 的 semi-join 優化
slug: mysql-exists-optimize
date: 2020-07-23T12:00:00+08:00
categories:
  - MySQL
tags:
  - "8.0"
  - 內核
weight: 1  
---

`MySQL5.6` 開始新增了 `semi-join` 優化了 `IN (SELECT ... FROM ...)`，但有相似效果的 `EXISTS` 及 `NOT IN/EXISTS` 卻沒有類似的優化，大多時候可能都必須透過其他的方式 (例如： `LEFT JOIN`) 來達到優化的效果，但從 `8.0.16`和 `8.0.17` 大家都能享受到優化，可以更愉快的使用這類語法囉！

## 實際執行

### 表結構

```sql
CREATE TABLE `member` (
  `id` int NOT NULL,
  `name` varchar(20) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
INSERT INTO member VALUES(1,'Johnson'),(2,'John'),(3,'Son');

CREATE TABLE `orders` (
  `id` int NOT NULL,
  `member_id` int NOT NULL,
  `amount` int NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
INSERT INTO orders VALUES(1,1,10),(2,2,20),(3,3,30);
```

### EXISTS 優化

`8.0.16` 中優化了 `EXISTS` 也可以吃到 `IN` 的 `semi-join` 優化！

```sql
-- 5.7.*
mysql> EXPLAIN SELECT * FROM orders WHERE EXISTS(SELECT * FROM member );
+----+-------------+--------+-------+---------+-------------+
| id | select_type | table  | type  | key     | Extra       |
+----+-------------+--------+-------+---------+-------------+
|  1 | PRIMARY     | orders | ALL   | NULL    | NULL        |
|  2 | SUBQUERY    | member | index | PRIMARY | Using index |
+----+-------------+--------+-------+---------+-------------+
2 rows in set, 1 warning (0.00 sec)

mysql> SHOW WARNINGS\G
*************************** 1. row ***************************
  Level: Note
   Code: 1003
Message: /* select#1 */ select `test`.`orders`.`id` AS `id`,
`test`.`orders`.`member_id` AS `member_id`,
`test`.`orders`.`amount` AS `amount`
from `test`.`orders` where 1
```

```sql
 -- 8.0.16
mysql> EXPLAIN SELECT * FROM orders WHERE EXISTS(SELECT * FROM member );
+----+-------------+--------+-------+---------+---------------------------------------+
| id | select_type | table  | type  | key     | Extra                                 |
+----+-------------+--------+-------+---------+---------------------------------------+
|  1 | SIMPLE      | member | index | PRIMARY | Using index; FirstMatch               |
|  1 | SIMPLE      | orders | ALL   | NULL    | Using join buffer (Block Nested Loop) |
+----+-------------+--------+-------+---------+---------------------------------------+
2 rows in set, 1 warning (0.00 sec)

mysql> SHOW WARNINGS\G
*************************** 1. row ***************************
  Level: Note
   Code: 1003
Message: /* select#1 */ select `test`.`orders`.`id` AS `id`,
`test`.`orders`.`member_id` AS `member_id`,
`test`.`orders`.`amount` AS `amount` 
from `test`.`orders` semi join (`test`.`member`) where 1
```

如上範例，可以注意到 `8.0.16` 的 `EXISTS` 語句不只在 `EXPLAIN` 中出現 `semi-join` 的 `FirstMatch` 優化，而且在 `WARNINGS` 中的改寫也出現 `semi-join`，相對的 `SUBQUERY`也消失了。

### NOT IN/EXISTS 優化

`8.0.17` 中優化了帶有 `NOT` 的 `IN、EXISTS` 可以使用 `anti-join` 的優化

```sql
-- 5.7.*、8.0.16
mysql> EXPLAIN SELECT count(*) FROM orders WHERE  NOT EXISTS 
(SELECT * FROM member WHERE orders.member_id = member.id);
+----+--------------------+--------+--------+---------+-----------------------+--------+-------------+
| id | select_type        | table  | type   | key     | ref                   | rows   | Extra       |
+----+--------------------+--------+--------+---------+-----------------------+--------+-------------+
|  1 | PRIMARY            | orders | ALL    | NULL    | NULL                  | 100536 | Using where |
|  2 | DEPENDENT SUBQUERY | member | eq_ref | PRIMARY | test.orders.member_id |      1 | Using index |
+----+--------------------+--------+--------+---------+-----------------------+--------+-------------+
2 rows in set, 1 warning (0.00 sec)

mysql> SHOW warnings\G
*************************** 1. row ***************************
  Level: Note
   Code: 1276
Message: Field or reference 'test.orders.member_id' of SELECT #2 was resolved in SELECT #1
*************************** 2. row ***************************
  Level: Note
   Code: 1003
Message: /* select#1 */ select count(0) AS `count(*)` from `test`.`orders` 
where (not(exists(/* select#2 */ select 1 from `test`.`member` 
where (`test`.`orders`.`member_id` = `test`.`member`.`id`))))

mysql> SELECT count(*) FROM orders WHERE  NOT EXISTS 
(SELECT * FROM member WHERE orders.member_id = member.id);
+----------+
| count(*) |
+----------+
|    99897 |
+----------+
1 row in set (0.25 sec)
```

```sql
-- 8.0.17
mysql> EXPLAIN SELECT count(*) FROM orders WHERE  NOT EXISTS 
(SELECT * FROM member WHERE orders.member_id = member.id);
+----+--------------+-------------+--------+---------------------+-----------------------+--------+-------------------------+
| id | select_type  | table       | type   | key                 | ref                   | rows   | Extra                   |
+----+--------------+-------------+--------+---------------------+-----------------------+--------+-------------------------+
|  1 | SIMPLE       | orders      | ALL    | NULL                | NULL                  | 100411 | NULL                    |
|  1 | SIMPLE       | <subquery2> | eq_ref | <auto_distinct_key> | test.orders.member_id |      1 | Using where; Not exists |
|  2 | MATERIALIZED | member      | index  | PRIMARY             | NULL                  |    100 | Using index             |
+----+--------------+-------------+--------+---------------------+-----------------------+--------+-------------------------+
2 rows in set, 1 warning (0.00 sec)

mysql> SHOW WARNINGS\G
*************************** 1. row ***************************
  Level: Note
   Code: 1276
Message: Field or reference 'test.orders.member_id' of SELECT #2 was resolved in SELECT #1
*************************** 2. row ***************************
  Level: Note
   Code: 1003
Message: /* select#1 */ select count(0) AS `count(*)` from `test`.`orders` 
anti join (`test`.`member`) on((`<subquery2>`.`id` = `test`.`orders`.`member_id`)) 
where true

mysql> SELECT count(*) FROM orders WHERE  NOT EXISTS 
(SELECT * FROM member WHERE orders.member_id = member.id);
+----------+
| count(*) |
+----------+
|    99906 |
+----------+
1 row in set (0.03 sec)
```

如上範例，可以注意到 `8.0.17`的 `NOT IN`能看到 `WARNING` 中被改寫為 `anti-join`，相對的 `SUBQUERY`也消失了，而且還沒有可怕的 `DEPENDENT SUBQUERY`，執行時間上當然也有所縮短。

## 結語：

本次 `MySQL 8.0` 針對 `(NOT) IN|EXISTS` 的優化方式個人認為相當的實用，並且 `semi(anti)-join` 還能夠搭配上 `8.0` 的`HASH JOIN` 相比過去能更大幅度的改善這類語法的效能，也不用像過去為了讓這類語法有更好的效能，用其他手段改寫導致可能犧牲一些可讀性，一舉數得真的很棒呢 😍

## 延伸閱讀

[semi-join(半連結)和anti-join(反連結)]({{< ref "post/MySQL/semi-join&anti-join/index.md" >}})

## 參考資料

[MySQL 5.7 文檔](https://dev.mysql.com/doc/refman/5.7/en/semijoins.html)

[MySQL 8.0 文檔](https://dev.mysql.com/doc/refman/8.0/en/semijoins.html)

[Antijoin in MySQL8(by MySQL Server Blog)](https://mysqlserverteam.com/antijoin-in-mysql-8/)

[anti-join幾點總結(by 知乎-知數堂)](https://zhuanlan.zhihu.com/p/99195571)