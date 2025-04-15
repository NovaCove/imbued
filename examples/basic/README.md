# basic imbued example

## Testing it out
Once `imbued` is installed and shell aliases are configured:

```sh
cd examples/basic
imbued client set-secret --name IMBUED_BASIC_EXAMPPLE_DATABASE_PASSWORD
cd ..
echo $DB_PASSWORD # Should be empty
cd basic
echo $DB_PASSWORD # Will be set to whatever password you set in line 2
```
