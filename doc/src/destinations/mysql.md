# MySQL data destination

The **MySQL** data destination allows you to write the unified users into a MySQL table and keep it synchronized.

MySQL is an open-source relational database management system. It's popular for web applications due to its scalability, security, and performance.

### On this page

* [Add a MySQL data destination](#add-a-mysql-data-destination)

### Add a MySQL data destination

1. From the Meergo admin, go to **Connections > Destinations**.
2. On the **Destinations** page, click **Add new destinations**.
3. Search for the **MySQL** destination; you can use the search bar at the top to help you.
4. Next to the **MySQL** destination, click the **+** icon.
5. On the `Add MySQL destination connection` page, in the **Name** field, enter a name for the destination to easily recognize it later.
6. In the remaining fields, provide the necessary information to access your MySQL instance:
    * **Host**: The name of the host.
    * **Port**: The port. The default is 3306.
    * **Username**: A username with read and write access to the table.
    * **Password**: The password for the user.
    * **Database name**: The name of the database.
7. Optional: Click **Test connection** to check if the inserted data is correct.
8. Click **Add**.

Once the MySQL data destination is added, the **Actions** page will be displayed, indicating the actions required to update the table.