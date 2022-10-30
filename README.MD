# apigw
This project aim to provide an api endpoint that execute multiple action and return result as json.

Like : 
 - I need to make a SQL Query to a DB, but we don't have a native api endpoint
 - I need to query a LDAP server
 - I want to get data from another api endpoint
 - I need to make a snmpquery 
 - I need to send a snmptrap
 - I need to merge some json data from different source

All definition of api endpoint and actions are defined in a json file.

So it's useless to change software code except for a new module.


# Configure

## Configuration file location
The configuration file can be in multiple location
./config
/etc/apigw/
/usr/local/etc/apigw

Copy the config.json file into one the possible location.

## Basic configuration
Edit it the file to configure :

 - the listening port of the http service (listening_port) 
 - the output file for logs (output_file) with the following values
   - stdout
   - path to a file

## Definition of an api endpoint
An api endpoint in apigw is a path that can be called by 3rd party software.
Actually it can be :
 - a sub api call to an external data source (basic no auth)
 - a sql query

In the future some new sub module will be added like :
 - api with authentication
 - ldap search
 - complex sql query with dynamic parameters
 - get snmp values
 - send snmptrap


## Create a new api endpoint to reach a third api endpoint

Edit the api json section.
Create a new json object with the wished endpoint path like **/test/api**
Define the type as **api**
Associate a source ie **catfact**
Define the http method (actually only get is supported) for your api call
Define the target http method for the target api that will be used as a data source

```
    "/test/api": {
      "type": "api",
      "source": "catfact",
      "method": "get",
      "target_method": "get"
    },
```

Then edit the **sources** section
Add a new object with the same name as your previously defined source ie: **catfact**
And add the url that will be used as a data source

```
    "catfact": {
        "url": "https://catfact.ninja/fact"
    }
```

## Create a new api endpoint to query a database

Edit the api json section.
Create a new json object with the wished endpoint path like **/users**
Define the type api as **sql**
Associate a source ie **mysql**
Define the http method (actually only get is supported) for your api call.
Define the sql query that need to be performed. 

```
    "/users": {
      "type": "sql",
      "source": "mysql",
      "method": "get",
      "query": "select * from users"
    },
```

Then edit the **sources** section
Add a new object with the same name as your previously defined source ie: **mysql**
And add the url that will be used as a data source


```
    "mysql": {
      "dbname": "test",
      "engine": "mysql",
      "host": "localhost",
      "port": "3306",
      "username": "test",
      "password": "test"
    },
```
