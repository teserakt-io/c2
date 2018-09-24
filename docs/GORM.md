
# GORM KB

##Miscellaneous notes

### GORM model declaration

The GORM model structure looks like this:

```
type User struct {
  gorm.Model
  Name         string
  Age          sql.NullInt64
  Birthday     *time.Time
}
```

`gorm.Model` implicitly declares an `id` field as an `Int`, with the properties 
unique, primary key, not null. There are also three date fields, `created_at`,
`updated_at` and `deleted_at`, which will be updated whenever any of the 
verb actions are performed.

This enables **soft delete**. However, gorm works _just fine_ without this 
provided a primary key is supplied - in this case deletes are **hard** - the 
row is properly dropped.

### Slice queries

Query syntax looks something like this:

    var instance ModelType
    db.Where(...).Selector(&instance)

One selector option is to use strings:

    "E4ID=?", byteslice

This seems to fail; however the struct form works:

    &instance{FieldName: byteslice}

I have not explored why.

