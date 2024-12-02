{% extends "/layouts/doc.html" %}
{% macro Title string %}MySQL data source{% end %}
{% Article %}

# MySQL data source

The **MySQL** data source allows you to read users from a MySQL database and unify them as users within Meergo.

MySQL is an open-source relational database management system. It's popular for web applications due to its scalability, security, and performance.

### On this page

* [Add a MySQL data source](#add-a-mysql-data-source)

### Add a MySQL data source

1. From the Meergo admin, go to **Connections > Sources**.
2. On the **Sources** page, click **Add new source**.
3. Search for the **MySQL** source; you can use the search bar at the top to help you.
4. Next to the **MySQL** source, click the **+** icon.
5. On the `Add MySQL source connection` page, in the **Name** field, enter a name for the source to easily recognize it later.
6. In the remaining fields, provide the necessary information to access your MySQL instance:
    * **Host**: The name of the host.
    * **Port**: The port. The default is 3306.
    * **Username**: A username with read access to the tables.
    * **Password**: The password for the user.
    * **Database name**: The name of the database.
7. Optional: Click **Test connection** to check if the inserted data is correct.
8. Click **Add**.

Once the MySQL data source is added, the **Actions** page will be displayed. This page indicates what actions to perform with the users read from MySQL.
