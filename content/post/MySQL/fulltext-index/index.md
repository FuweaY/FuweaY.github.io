---
title: MySQL FullText Index(全文檢索)
description: 介紹 MySQL FullText Index
slug: mysql-fulltext-index
date: 2024-12-01T12:00:00+08:00
categories:
   - MySQL
weight: 1  
---
# FullText Index(全文檢索)

ngrame 全文解析器

MySQL 從 5.7.6 開始內建 ngrame 全文解析器，用來支援中文、日文、韓文分詞

## InnoDB Full-Text Index

### inverted index(倒排索引)

inverted index 經常被用於全文檢索，MySQL 的 Full-Text Index 也是基於 inverted index 來實現的。

inverted index 將文檔中的不重複單詞構成一個列表，每一個單詞都會記錄包含此單詞的文檔列表。

```sql
CREATE TABLE small_test(
  `id` int unsigned NOT NULL AUTO_INCREMENT COMMENT '流水號',
  `content` varchar(300) NOT NULL COMMENT '內文',
  PRIMARY KEY (`id`),
  FULLTEXT KEY `content_index` (`content`) /*!50100 WITH PARSER `ngram` */  
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci; 

INSERT INTO small_test(content) VALUES ('我是強森'),('這是測試'),('強森是我');

SET GLOBAL innodb_ft_aux_table = 'johnson/small_test';

SELECT * FROM information_schema.INNODB_FT_INDEX_CACHE;
+--------+--------------+-------------+-----------+--------+----------+
| WORD   | FIRST_DOC_ID | LAST_DOC_ID | DOC_COUNT | DOC_ID | POSITION |
+--------+--------------+-------------+-----------+--------+----------+
| 強森   |            2 |           4 |         2 |      2 |        6 |
| 強森   |            2 |           4 |         2 |      4 |        0 |
| 我是   |            2 |           2 |         1 |      2 |        0 |
| 是強   |            2 |           2 |         1 |      2 |        3 |
| 是我   |            4 |           4 |         1 |      4 |        6 |
| 是測   |            3 |           3 |         1 |      3 |        3 |
| 森是   |            4 |           4 |         1 |      4 |        3 |
| 測試   |            3 |           3 |         1 |      3 |        6 |
| 這是   |            3 |           3 |         1 |      3 |        0 |
+--------+--------------+-------------+-----------+--------+----------+
```

### InnoDB Full-Text Index Tables

在建立 Full-Text Index 之後，還會出現以下 table：

```bash
-rw-r-----.  1 mysql mysql     114688 Jun 30 09:05 fts_00000000000004a6_0000000000000129_index_1.ibd
-rw-r-----.  1 mysql mysql     114688 Jun 30 09:05 fts_00000000000004a6_0000000000000129_index_2.ibd
-rw-r-----.  1 mysql mysql     114688 Jun 30 09:05 fts_00000000000004a6_0000000000000129_index_3.ibd
-rw-r-----.  1 mysql mysql     114688 Jun 30 09:05 fts_00000000000004a6_0000000000000129_index_4.ibd
-rw-r-----.  1 mysql mysql     114688 Jun 30 09:05 fts_00000000000004a6_0000000000000129_index_5.ibd
-rw-r-----.  1 mysql mysql     114688 Jun 30 09:05 fts_00000000000004a6_0000000000000129_index_6.ibd
-rw-r-----.  1 mysql mysql     114688 Jun 30 09:05 fts_00000000000004a6_being_deleted_cache.ibd
-rw-r-----.  1 mysql mysql     114688 Jun 30 09:05 fts_00000000000004a6_being_deleted.ibd
-rw-r-----.  1 mysql mysql     114688 Jun 30 09:05 fts_00000000000004a6_config.ibd
-rw-r-----.  1 mysql mysql     114688 Jun 30 09:05 fts_00000000000004a6_deleted_cache.ibd
-rw-r-----.  1 mysql mysql     114688 Jun 30 09:05 fts_00000000000004a6_deleted.ibd
```

可以分為 4 類－

1. index_1~6：數量固定會有 6 個文件用於儲存倒排索引，儲存的是 words、position 和 DOC_ID，會根據 word 進行排序並分區映射到不同的文件中。
   倒排索引分為 6 個表用來支持 parallel index creation，預設為 2 個線程，當在大表創建 Full-Text Index 時，可以透過 [innodb_ft_sort_pll_degree](https://dev.mysql.com/doc/refman/8.0/en/innodb-parameters.html#sysvar_innodb_ft_sort_pll_degree) 來調大併行線程數。
2. deleted：包含資料已經刪除的 DOC_ID，但還沒從 Full-Text index 中刪除。
   deleted_cache：為前者的內存緩存。
3. being_deleted：包含資料已經刪除的 DOC_ID，且當前正在從 Full-Text index 中刪除。
   being_deleted_cache：為前者的內存緩存。
4. config：包含 Full-Text index 的內部狀態資訊，最重要的是其中有儲存 FTS_SYNCED_DOC_ID，該值用來記錄解析被寫到 disk 的 DOC_ID，在崩潰回覆時會根據這個值判斷哪些 DOC 需要重新解析並將其添加到 Full-text index cache，查詢 INFORMATION_SCHEMA.INNODB_FT_CONFIG 可以查看其中的資料。

## InnoDB Full-Text Index Cache

## InnoDB Full-Text Index DOC_ID and FTS_DOC_ID Column

InnoDB 需要透過 DOC_ID 來對應 Word 所在的紀錄，因此建立 Full-Text Index 的時候，也會在表上隱式建立一個 FTS_DOC_ID 的欄位，請參考以下事例：

```sql
mysql> CREATE TABLE small_test_2(
    ->   `id` int unsigned NOT NULL AUTO_INCREMENT COMMENT '流水號',
    ->   `content` varchar(300) NOT NULL COMMENT '內文',
    ->   PRIMARY KEY (`id`)
    -> ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
Query OK, 0 rows affected (0.03 sec)

mysql> ALTER TABLE small_test_2 ADD FULLTEXT KEY `content_index` (`content`) /*!50100 WITH PARSER `ngram` */;
Query OK, 0 rows affected, 1 warning (0.11 sec)
Records: 0  Duplicates: 0  Warnings: 1

mysql> show warnings\G
*************************** 1. row ***************************
  Level: Warning
   Code: 124
Message: InnoDB rebuilding table to add column FTS_DOC_ID

mysql> show extended columns from small_test_2;
+-------------+--------------+------+-----+---------+----------------+
| Field       | Type         | Null | Key | Default | Extra          |
+-------------+--------------+------+-----+---------+----------------+
| id          | int unsigned | NO   | PRI | NULL    | auto_increment |
| content     | varchar(300) | NO   | MUL | NULL    |                |
| FTS_DOC_ID  |              | NO   |     | NULL    |                |
| DB_TRX_ID   |              | NO   |     | NULL    |                |
| DB_ROLL_PTR |              | NO   |     | NULL    |                |
+-------------+--------------+------+-----+---------+----------------+
5 rows in set (0.00 sec)
```

因此在該表上建立第一個 Full-Text Index 需要重建表，如果想要避免重建表可以在 CREATE TABLE  時預先建立此欄位，如下事例：

```sql
mysql> CREATE TABLE small_test_3(
    ->   `FTS_DOC_ID` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    ->   `content` varchar(300) NOT NULL COMMENT '內文',
    ->   PRIMARY KEY (`FTS_DOC_ID`)
    -> ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
Query OK, 0 rows affected (0.12 sec)

mysql> alter table small_test_3 add FULLTEXT KEY `content_index` (`content`) /*!50100 WITH PARSER `ngram` */;
Query OK, 0 rows affected (0.36 sec)
Records: 0  Duplicates: 0  Warnings: 0
```

刪除 Full-Text Index 時，為了避免下次建立時需重建表，因此不會移除 FTS_DOC_ID 欄位。

## InnoDB Full-Text Index delete handling

為了避免 delete 原表資料時，在索引表中出現大量的 delete 引發資源爭用的問題，已刪除的 DOC_ID 會被記錄在 deleted 表中且不會在索引表中移出，而是會在返回查詢結果之前，透過 deleted 表來過濾已刪除的 DOC_ID。可以透過設置 [innodb_optimize_fulltext_only](https://dev.mysql.com/doc/refman/8.0/en/innodb-parameters.html#sysvar_innodb_optimize_fulltext_only) = ON 後執行 OPTIMIZE TABLE 來重建 Full-Text Index 來移除已刪除 DOC_ID 的索引，這些會被轉存到 being_deleted 檔案中。

## InnoDB Full-Text Index Transaction Handling

## InnoDB Full-Text Indexes 相關 TABLE

information_schema 下有提供觀察 Full-Text Index 的表，在使用前須要先設定 [innodb_ft_aux_table](https://dev.mysql.com/doc/refman/8.0/en/innodb-parameters.html#sysvar_innodb_ft_aux_table) 調整為想要觀察的表，否則除了 INNODB_FT_DEFAULT_STOPWORD 以外都會是 empty。

```sql
SET GLOBAL innodb_ft_aux_table = 'database_name/table_name';
```

共有以下 6 張表－

- INNODB_FT_CONFIG：
- INNODB_FT_INDEX_TABLE
- INNODB_FT_INDEX_CACHE
- INNODB_FT_DEFAULT_STOPWORD
- INNODB_FT_DELETED
- INNODB_FT_BEING_DELETED

## 全文檢索模式

### IN NATURAL LANGUAGE MODE (自然語言模式)

此為默認的方式

```sql
mysql> select * from small_test;
+----+-----------------------------+
| id | content                     |
+----+-----------------------------+
|  1 | MySQL 的全文檢索測試        |
|  2 | MySQL vs Solr               |
|  3 | Solr 的全文檢索             |
|  4 | ES 的全文檢索               |
+----+-----------------------------+
4 rows in set (0.00 sec)

mysql> SELECT * FROM small_test
    -> WHERE MATCH(`content`)
    -> AGAINST ('MySQL');
+----+-----------------------------+
| id | content                     |
+----+-----------------------------+
|  1 | MySQL 的全文檢索測試        |
|  2 | MySQL vs Solr               |
+----+-----------------------------+
2 rows in set (0.01 sec)

```

### IN Boolean

```sql
mysql> select * from small_test;
+----+-----------------------------+
| id | content                     |
+----+-----------------------------+
|  1 | MySQL 的全文檢索測試        |
|  2 | MySQL vs Solr               |
|  3 | Solr 的全文檢索             |
|  4 | ES 的全文檢索               |
+----+-----------------------------+
4 rows in set (0.00 sec)

mysql> SELECT * FROM small_test
    -> WHERE MATCH(`content`)
    -> AGAINST ('+Mysql -Solr' IN BOOLEAN MODE);
+----+-----------------------------+
| id | content                     |
+----+-----------------------------+
|  1 | MySQL 的全文檢索測試        |
+----+-----------------------------+
1 row in set (0.00 sec)

```

### With Query Expansion (查詢擴展)

```sql
mysql> SELECT * FROM small_test;
+----+-----------------------------+
| id | content                     |
+----+-----------------------------+
|  1 | MySQL 的全文檢索測試        |
|  2 | MySQL vs Solr               |
|  3 | Solr 的全文檢索             |
|  4 | ES 的全文檢索               |
|  5 | 我是無辜的                  |
+----+-----------------------------+
5 rows in set (0.00 sec)

mysql> SELECT *
    -> FROM small_test
    -> WHERE MATCH(`content`)
    -> AGAINST ('MySQL' WITH QUERY EXPANSION);
+----+-----------------------------+
| id | content                     |
+----+-----------------------------+
|  1 | MySQL 的全文檢索測試        |
|  2 | MySQL vs Solr               |
|  3 | Solr 的全文檢索             |
|  4 | ES 的全文檢索               |
+----+-----------------------------+
4 rows in set (0.00 sec)

```

# 限制

有切 partition 的表不支援 FullText index

# 參考

[MySQL :: MySQL 8.0 Reference Manual :: 15.6.2.4 InnoDB Full-Text Indexes](https://dev.mysql.com/doc/refman/8.0/en/innodb-fulltext-index.html)

[MySQL :: MySQL 8.0 Reference Manual :: 12.10 Full-Text Search Functions](https://dev.mysql.com/doc/refman/8.0/en/fulltext-search.html)

[http://mysql.taobao.org/monthly/2015/10/01/](http://mysql.taobao.org/monthly/2015/10/01/)

[https://iter01.com/523515.html](https://iter01.com/523515.html)

[https://database.51cto.com/art/202010/630055.htm](https://database.51cto.com/art/202010/630055.htm)

[mysql中文全文检索从入门到放弃_弹指天下-CSDN博客](https://blog.csdn.net/w1014074794/article/details/106746114)