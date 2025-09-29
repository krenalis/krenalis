{% extends "/layouts/doc.html" %}
{% macro Title string %}MySQL (Destination){% end %}
{% Article %}

# MySQL (Destination)

The destination connector for MySQL allows you to write the unified users into a MySQL table and keep it synchronized.

MySQL is an open-source relational database management system. It's popular for web applications due to its scalability, security, and performance.

### On this page

* [Add destination connection for MySQL](#add-destination-connection-for-mysql)
* [Export users to MySQL](#export-users-to-mysql)

### Add destination connection for MySQL

1. From the Meergo Admin console, go to **Connections > Destinations**.
2. On the **Destinations** page, click **Add a new destination ⊕**.
3. Search **MySQL**; you can use the search bar at the top or filter by category.
4. Click on the connector for **MySQL**. A panel will open on the right.
5. Click on **Add destination...**. The `Add destination connection for MySQL` page will appear.
6. In the **Name** field, enter a name for the destination to easily recognize it later.
7. In the remaining fields, provide the necessary information to access your MySQL instance:
    * **Host**: The name of the host.
    * **Port**: The port. The default is 3306.
    * **Username**: A username with read and write access to the table.
    * **Password**: The password for the user.
    * **Database name**: The name of the database.
8. (Optional) Click **Test connection** to check if the inserted data is correct.
9. Click **Add**.

Once the destination connection for MySQL is added, the **Actions** page will be displayed.

**Note about choosing the table key**: when exporting to a destination connection for  MySQL, it is necessary for the table key selected on the action screen to match the primary key of the table to which you intend to export.

### Export users to MySQL

1. From the Meergo Admin console, go to **Connections > Destinations**.
2. Click on the destination connection for MySQL where you want to export the users.
3. If there are no actions, click  **Add**, otherwise click **Add new action ⊕**.
4. Enter the name of the MySQL table where users should be added or updated.
5. Click **Confirm** to proceed.
6. Specify the key column. This column will be used to identify and update existing rows.
7. Define the mapping or use a transformation function to convert the users in your workspace's data warehouse into MySQL rows.
8. Click **Add**.
