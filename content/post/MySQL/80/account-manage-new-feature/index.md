---
title: MySQL8.0 新增的帳號管理功能
description: 介紹 MySQL8.0 新增的帳號管理功能
slug: mysql-account-manage-new-feature
date: 2021-07-12T12:00:00+08:00
categories:
   - MySQL
tags:
   - "8.0"
   - 功能
weight: 1  
---
## Password Verification-Required Policy

MySQL 8.0.13 開始，可以設定在修改密碼時需要一併提供舊密碼。

可透過以下 2 種方式設定 -

1. 全域系統變數 [password_require_current](https://dev.mysql.com/doc/refman/8.0/en/server-system-variables.html#sysvar_password_require_current)  預設值為 OFF ，可以透過調整為 ON 要求修改密碼時提供就密碼。
2. CREATE 或 ALTER USER 時加上 PASSWORD REQUIRE CURRENT [DEFAULT | OPTIONAL] 設定。

    ```sql
    -- admin 建立帳號
    mysql> CREATE USER test@localhost IDENTIFIED BY '123' PASSWORD REQUIRE CURRENT;
    Query OK, 0 rows affected (0.00 sec)
    
    -- user 修改密碼
    mysql> ALTER USER test@localhost IDENTIFIED BY '123';
    ERROR 3892 (HY000): Current password needs to be specified in the REPLACE clause in order to change it.
    
    mysql> ALTER USER test@localhost IDENTIFIED BY '123' REPLACE '456';
    Query OK, 0 rows affected (0.01 sec)
    ```

   PASSWORD REQUIRE CURRENT：密碼更改必須指定當前密碼。

   PASSWORD REQUIRE CURRENT OPTIONAL：密碼更改不強迫指定當前密碼。

   PASSWORD REQUIRE CURRENT DEFAUT：依照全域系統變數 [password_require_current](https://dev.mysql.com/doc/refman/8.0/en/server-system-variables.html#sysvar_password_require_current) 設定。

> 💡 NOTE：當 USER 具有 global create user 權限或者是在 mysql database 具有 update 權限時，則不受任何限制，意即不需要提供舊密碼。

參考

[https://dev.mysql.com/doc/refman/8.0/en/password-management.html#password-reverification-policy](https://dev.mysql.com/doc/refman/8.0/en/password-management.html#password-reverification-policy)

[https://dev.mysql.com/doc/refman/8.0/en/server-system-variables.html#sysvar_password_require_current](https://dev.mysql.com/doc/refman/8.0/en/server-system-variables.html#sysvar_password_require_current)

## 雙密碼功能 (Dual Passwords)

MySQL 8.0.14 開始在帳戶管理新增了 2 個子句，用來提供雙密碼的功能，這樣就可以分階段且不需各單位配合也不需要停機的況下更換密碼。

### RETAIN CURRENT PASSWORD

- 將 user 當前(舊)密碼替換為第二(secondary)密碼，新的密碼則會成為主(primary)密碼。

    ```sql
    mysql> CREATE USER test@localhost IDENTIFIED BY '123';
    Query OK, 0 rows affected (0.01 sec)
    
    mysql> ALTER USER test@localhost IDENTIFIED BY '456' RETAIN CURRENT PASSWORD;
    Query OK, 0 rows affected (0.04 sec)
    
    -- 使用 secondary(舊) 密碼登入成功
    [root@localhost ~]$ mysql -utest -p123
    Your MySQL connection id is 82583
    Server version: 8.0.21 MySQL Community Server - GPL
    
    -- 使用 primary(新) 密碼登入成功
    [root@localhost ~]$ mysql -utest -p456
    Your MySQL connection id is 82584
    Server version: 8.0.21 MySQL Community Server - GPL
    ```

- 新舊密碼為空時，無法指定 RETAIN CURRENT PASSWORD。

    ```sql
    mysql> CREATE USER test@localhost;
    Query OK, 0 rows affected (0.01 sec)
    
    mysql> ALTER USER test@localhost identified by '123' RETAIN CURRENT PASSWORD;
    ERROR 3878 (HY000): Empty password can not be retained as second password for user 'test'@'localhost'.
    
    mysql> ALTER USER test@localhost identified by '123';
    Query OK, 0 rows affected (0.02 sec)
    
    mysql> ALTER USER test@localhost identified by '' RETAIN CURRENT PASSWORD;
    ERROR 3895 (HY000): Current password can not be retained for user 'test'@'localhost' because new password is empty.
    ```

- 當 user 已有第二(secondary)密碼時，在未指定 RETAIN CURRENT PASSWORD 的情況下更改主(primary)密碼，第二(secondary)密碼會維持不變。

    ```sql
    mysql> CREATE USER test@localhost IDENTIFIED BY '123';
    Query OK, 0 rows affected (0.00 sec)
    
    mysql> ALTER USER test@localhost IDENTIFIED BY '456' RETAIN CURRENT PASSWORD;
    Query OK, 0 rows affected (0.00 sec)
    -- 此時 123 為 secondary password， 456 為 parmary password
    
    mysql> ALTER USER test@localhost IDENTIFIED BY '789';
    Query OK, 0 rows affected (0.00 sec)
    -- 此時 123 為 secondary password， 789 為 parmary password
    
    -- 使用 123 登入成功
    [root@localhost ~]$ mysql -utest -p123
    Your MySQL connection id is 82583
    Server version: 8.0.21 MySQL Community Server - GPL
    
    -- 使用 456 登入失敗
    [root@localhost ~]$ mysql -utest -p456
    ERROR 1045 (28000): Access denied for user 'test'@'localhost' (using password: YES)
    
    -- 使用 789 登入成功
    [root@localhost ~]$ mysql -utest -p789
    Your MySQL connection id is 82583
    Server version: 8.0.21 MySQL Community Server - GPL
    
    mysql> ALTER USER test@localhost IDENTIFIED BY 'abc' RETAIN CURRENT PASSWORD;
    Query OK, 0 rows affected (0.00 sec)
    -- 此時 789 為 secondary password， abc 為 parmary password
    ```

- 當更改身分驗證插件時，無法指定 RETAIN CURRENT PASSWORD，並且會將第二(secondary)密碼丟棄。

### DISCARD OLD PASSWORD

刪除第二(secondary)密碼。

```sql
ALTER USER test@localhost DISCARD OLD PASSWORD;

-- 使用 secondary password 登入
[root@localhost ~]$ mysql -utest -p123
ERROR 1045 (28000): Access denied for user 'test'@'localhost' (using password: YES)
```

參考

[https://dev.mysql.com/doc/refman/8.0/en/alter-user.html](https://dev.mysql.com/doc/refman/8.0/en/alter-user.html)

[https://dev.mysql.com/doc/refman/8.0/en/password-management.html#dual-passwords](https://dev.mysql.com/doc/refman/8.0/en/password-management.html#dual-passwords)

[https://www.percona.com/blog/using-mysql-8-dual-passwords/](https://www.percona.com/blog/using-mysql-8-dual-passwords/)

## 生成隨機密碼(Random Password Generation)

MySQL 8.0.18 開始在設定密碼的時候可以使用 RANDOM PASSWORD 來為 USER 生成隨機密碼。

```sql
mysql> CREATE USER test@localhost IDENTIFIED BY RANDOM PASSWORD;
+------+-----------+----------------------+
| user | host      | generated password   |
+------+-----------+----------------------+
| test | localhost | Oa-J)+nSI40_!TqIaYIt |
+------+-----------+----------------------+

mysql> ALTER USER test@localhost IDENTIFIED BY RANDOM PASSWORD;
+------+-----------+----------------------+
| user | host      | generated password   |
+------+-----------+----------------------+
| test | localhost | XgtPCz>Sx)Yf8Sxhpg:D |
+------+-----------+----------------------+
1 row in set (0.02 sec)
```

生成的密碼長度由系統變數 [generated_random_password_length](https://dev.mysql.com/doc/refman/8.0/en/server-system-variables.html#sysvar_generated_random_password_length) 決定，預設值為 20。

參考

[https://dev.mysql.com/doc/refman/8.0/en/password-management.html#random-password-generation](https://dev.mysql.com/doc/refman/8.0/en/password-management.html#random-password-generation)

[https://dev.mysql.com/doc/refman/8.0/en/server-system-variables.html#sysvar_generated_random_password_length](https://dev.mysql.com/doc/refman/8.0/en/server-system-variables.html#sysvar_generated_random_password_length)

## 登入失敗追蹤和帳號暫時鎖定(Failed-Login Tracking and Temporary Account Locking)

MySQL 8.0.19 開始，可以設定當該帳號連續輸入錯誤的密碼時，暫時將帳號鎖定。

- FAILED_LOGIN_ATTEMPTS N：表示當連續輸入 N 次錯誤密碼時，將會觸發鎖定。
- PASSWORD_LOCK_TIME {N | UNBOUNDED}：表示要鎖定 N 天，其中 UNBOUNDED 表示永久鎖定直到被解鎖。

以上 N 的允許值為 0~32767，其中 0 表示禁用，預設值皆為 0。只有當兩個 N 都不為 0 ，才能使用到此功能。

```sql
# 建立帳號，並設定當密碼連續輸入錯誤 3次時，則會鎖定 3 天
mysql> CREATE USER test@localhost IDENTIFIED BY '123'
    ->   FAILED_LOGIN_ATTEMPTS 3 PASSWORD_LOCK_TIME 3;
Query OK, 0 rows affected (0.02 sec)

[root@localhost ~]$ mysql -utest -p
Enter password:
ERROR 1045 (28000): Access denied for user 'test'@'localhost' (using password: YES)
[root@localhost ~]$ mysql -utest -p
Enter password:
ERROR 1045 (28000): Access denied for user 'test'@'localhost' (using password: YES)
[root@localhost ~]$ mysql -utest -p
Enter password:
ERROR 3955 (HY000): Access denied for user 'test'@'localhost'. 
Account is blocked for 3 day(s) (3 day(s) remaining) due to 3 consecutive failed logins.

-- error log
2021-07-07T08:13:23.364124Z 86982 [Note] [MY-010914] [Server] Access denied for user 'test'@'localhost'. 
Account is blocked for 3 day(s) (3 day(s) remaining) due to 3 consecutive failed logins.
```

以下方式可以重置計數並解鎖所有帳號：

- 重啟 server
- 執行 FLUSH PRIVILEGES

以下狀況會重置計數或解鎖個別帳號：

- 成功登入
- 持續鎖定的時間已過
- 使用 ALTER USER 變更鎖定設定，或者是使用 ACCOUNT UNLOCK 語句。

    ```sql
    # 變更鎖定設定也會重置
    mysql> ALTER USER test@localhost FAILED_LOGIN_ATTEMPTS 3 PASSWORD_LOCK_TIME 1;
    Query OK, 0 rows affected (0.02 sec)
    
    # 使用 ALTER USER ... ACCOUNT UNLOCK 解鎖
    mysql> ALTER USER 'test'@'localhost' ACCOUNT UNLOCK;
    Query OK, 0 rows affected (0.00 sec)
    ```


參考

[https://dev.mysql.com/doc/refman/8.0/en/password-management.html#failed-login-tracking](https://dev.mysql.com/doc/refman/8.0/en/password-management.html#failed-login-tracking)
