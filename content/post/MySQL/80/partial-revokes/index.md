---
title: MySQL8.0.16 新增 partial revokes 功能
description: 介紹 MySQL8.0.16 新增回收部分權限(partial revokes)的功能
slug: mysql-partial-revokes
date: 2021-03-12T12:00:00+08:00
categories:
  - MySQL
tags:
  - "8.0"
  - 功能
weight: 1  
---
在 MySQL 8.0.16 之前，如果我們要給予 USER 除了某個 DATABASE 以外都有權限，只能夠每個 DATABASE 都 GRANT 一次權限，這方面有時候並不是很方便。例如：總共有 100 個 DATABASE 其中只有 1 個不給予 USER 權限，我們就需要執行 99 個 GRANT 語法。

從 MySQL 8.0.16 開始，推出了可以回收部分權限 (Partial Revokes) 的功能，將粗粒度的 GRANT 權限用 REVOKE 收回一部分細粒度權限，大大增加了這方面需求的方便性。

## 展示

目標是給予 USER 除了 mysql.* 以外的所有權限，請查看以下範例：

- 當 MySQL 版本低於 8.0.16 或者關閉 partial revokes 功能，我們只能逐一給予其他 DB 權限：

    ```sql
    -- BEFORE MySQL 8.0.16，或關閉 partial revokes 功能
    mysql> SHOW GLOBAL VARIABLES LIKE 'partial_revokes';
    +-----------------+-------+
    | Variable_name   | Value |
    +-----------------+-------+
    | partial_revokes | OFF   |
    +-----------------+-------+
    1 row in set (0.00 sec)
    
    mysql> SHOW GRANTS FOR test@localhost;
    +-------------------------------------------+
    | Grants for test@localhost                 |
    +-------------------------------------------+
    | GRANT SELECT ON *.* TO `test`@`localhost` |
    +-------------------------------------------+
    1 row in set (0.00 sec)
    
    mysql> REVOKE SELECT ON mysql.* FROM test@localhost;
    ERROR 1141 (42000): There is no such grant defined for user 'test' on host 'localhost'
    
    mysql> REVOKE SELECT ON *.* FROM test@localhost;
    Query OK, 0 rows affected (0.00 sec)
    mysql> GRANT SELECT ON test.* TO test@localhost;
    Query OK, 0 rows affected (0.00 sec)
    mysql> GRANT SELECT ON performance_schema.* TO test@localhost;
    Query OK, 0 rows affected (0.00 sec)
    mysql> GRANT SELECT ON sys.* TO test@localhost;
    Query OK, 0 rows affected (0.01 sec) 
    
    mysql>  SHOW GRANTS FOR test@localhost;
    +--------------------------------------------------------------+
    | Grants for test@localhost                                    |
    +--------------------------------------------------------------+
    | GRANT USAGE ON *.* TO `test`@`localhost`                     |
    | GRANT SELECT ON `sys`.* TO `test`@`localhost`                |
    | GRANT SELECT ON `test`.* TO `test`@`localhost`               |
    | GRANT SELECT ON `performance_schema`.* TO `test`@`localhost` |
    +--------------------------------------------------------------+
    5 rows in set (0.00 sec)
    ```

- 當 MySQL 版本不小於 8.0.16 且開啟 partial revokes 功能，就能很方便的完成需求：

    ```sql
    -- AFTER MySQL 8.0.16，並開啟 partial revokes 功能
    mysql> SET GLOBAL partial_revokes = ON;
    Query OK, 0 rows affected (0.00 sec)
    
    mysql> SHOW GLOBAL VARIABLES LIKE 'partial_revokes';
    +-----------------+-------+
    | Variable_name   | Value |
    +-----------------+-------+
    | partial_revokes | ON    |
    +-----------------+-------+
    1 row in set (0.00 sec)
    
    mysql> REVOKE SELECT ON mysql.* FROM `test`@`localhost`;
    Query OK, 0 rows affected (0.00 sec)
    
    mysql> SHOW GRANTS FOR test@localhost;
    +----------------------------------------------------+
    | Grants for test@localhost                          |
    +----------------------------------------------------+
    | GRANT SELECT ON *.* TO `test`@`localhost`          |
    | REVOKE SELECT ON `mysql`.* FROM `test`@`localhost` |
    +----------------------------------------------------+
    2 rows in set (0.00 sec)
    ```


## 參考

[MySQL 文檔 - GRANT Statement](https://dev.mysql.com/doc/refman/8.0/en/grant.html)