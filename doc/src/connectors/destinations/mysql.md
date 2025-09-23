{% extends "/layouts/doc.html" %}
{% macro Title string %}MySQL data destination{% end %}
{% Article %}

# MySQL data destination

The **MySQL** data destination allows you to write the unified users into a MySQL table and keep it synchronized.

MySQL is an open-source relational database management system. It's popular for web applications due to its scalability, security, and performance.

### On this page

* [Add a MySQL data destination](#add-a-mysql-data-destination)
* [Export users to MySQL](#export-users-to-mysql)

### Add a MySQL data destination

1. From the Meergo Admin console, go to **Connections > Destinations**.
2. On the **Destinations** page, click **Add a new destination ⊕**.
3. Search for the **MySQL** destination; you can use the search bar at the top or filter by category.
4. Click on the **MySQL** connector. A panel will open on the right with information about **MySQL**.
5. Click on **Add destination**. The `Add MySQL destination connection` page will appear.
6. In the **Name** field, enter a name for the destination to easily recognize it later.
7. In the remaining fields, provide the necessary information to access your MySQL instance:
    * **Host**: The name of the host.
    * **Port**: The port. The default is 3306.
    * **Username**: A username with read and write access to the table.
    * **Password**: The password for the user.
    * **Database name**: The name of the database.
8. (Optional) Click **Test connection** to check if the inserted data is correct.
9. Click **Add**.

Once the MySQL data destination is added, the **Actions** page will be displayed.

**Note about choosing the table key**: when exporting to a MySQL data destination, it is necessary for the table key selected on the action screen to match the primary key of the table to which you intend to export.

### Export users to MySQL

1. From the Meergo Admin console, go to **Connections > Destinations**.
2. Click on the MySQL data destination where you want to export the users.
3. If there are no actions, click  **Add**, otherwise click **Add new action ⊕**.
4. Enter the name of the MySQL table where users should be added or updated.
5. Click **Confirm** to proceed.
6. Specify the key column. This column will be used to identify and update existing rows.
7. Define the mapping or use a transformation function to convert the users in your workspace's data warehouse into MySQL rows.
8. Click **Add**.
