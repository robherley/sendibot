schema "main" {
}

table "terms" {
  schema = schema.main
  column "id" {
    type = text
  }
  column "en" {
    type = text
  }
  column "jp" {
    type = text
  }
  primary_key {
    columns = [column.id]
  }
  index "idx_en" {
    columns = [column.en]
    unique = true
  }
}

table "subscriptions" {
  schema = schema.main
  column "id" {
    type = text
  }
  column "user_id" {
    type = text
  }
  column "term_id" {
    type = text
  }
  column "last_notified_at" {
    type = datetime
  }
  column "shops" {
    type = int
  }
  primary_key {
    columns = [column.id]
  }
  index "idx_last_notified_at" {
    columns = [column.last_notified_at]
  }
  index "idx_user_id_term_id" {
    columns = [column.user_id, column.term_id]
    unique = true
  }
}
