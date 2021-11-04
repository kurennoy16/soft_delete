# Soft Delete

This plugin supports `bool` type for `gorm` soft delete plugin. The original library works only with int.

```go
import "gorm.io/plugin/soft_delete"

type User struct {
  ID        uint
  Name      string
  DeletedAt soft_delete.IsDeleted
}

// Query
SELECT * FROM users WHERE is_deleted = FALSE;

// Delete
UPDATE users SET is_deleted = TRUE WHERE ID = 1;
```


## Updates DeletedAt field

Plugin searches for tag `DeletedAtField`, it's value uses as the field name.

```go
type User struct {
  ID        uint
  Name      string
  DeletedAt time.Time
  IsDel     soft_delete.DeletedAt `gorm:"DeletedAtField:DeletedAt"`
}

// Query
SELECT * FROM users WHERE is_del = FALSE;

// Delete
UPDATE users SET is_del = TRUE, deleted_at = /* current unix second */ WHERE ID = 1;
```
