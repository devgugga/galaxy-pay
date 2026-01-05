package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// IdempotencyKey holds the schema definition for the IdempotencyKey entity.
type IdempotencyKey struct {
	ent.Schema
}

// Fields of the IdempotencyKey.
func (IdempotencyKey) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.String("key").
			Unique().
			NotEmpty().
			MaxLen(255).
			Comment("Idempotency key from client (UUID)"),
		field.String("payment_id").
			Optional().
			Nillable().
			MaxLen(255).
			Comment("ID of the payment created (if applicable)"),
		field.Int("status_code").
			Default(200).
			Comment("HTTP status code of the response"),
		field.String("request_hash").
			Optional().
			MaxLen(64).
			Comment("Hash of the request body for validation"),
		field.Bytes("response_body").
			Comment("Serialized response body"),
		field.JSON("response_headers", map[string]string{}).
			Default(map[string]string{}).
			Comment("Response headers as JSON"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("Timestamp when the key was created"),
		field.Time("expires_at").
			Comment("Timestamp when the key expires"),
	}
}

// Indexes of the IdempotencyKey.
func (IdempotencyKey) Indexes() []ent.Index {
	return []ent.Index{
		// Unique index on key (already enforced by Unique() but explicit for clarity)
		index.Fields("key").Unique(),
		// Index on expires_at for efficient cleanup queries
		index.Fields("expires_at"),
		// Index on payment_id for related queries
		index.Fields("payment_id"),
		// Composite index for common query patterns
		index.Fields("key", "expires_at"),
	}
}
