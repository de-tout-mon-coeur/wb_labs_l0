package models

import "time"

type Order struct {
    OrderUID string                 `json:"order_uid"`
    TrackNumber string              `json:"track_number"`
    Entry string                    `json:"entry,omitempty"`
    Delivery map[string]interface{} `json:"delivery,omitempty"`
    Payment map[string]interface{}  `json:"payment,omitempty"`
    Items []map[string]interface{}  `json:"items,omitempty"`
    Locale string                    `json:"locale,omitempty"`
    InternalSignature string         `json:"internal_signature,omitempty"`
    CustomerID string               `json:"customer_id,omitempty"`
    DeliveryService string          `json:"delivery_service,omitempty"`
    Shardkey string                 `json:"shardkey,omitempty"`
    SmID int                        `json:"sm_id,omitempty"`
    DateCreated string              `json:"date_created,omitempty"`
    OofShard string                 `json:"oof_shard,omitempty"`

    Raw map[string]interface{}      `json:"-"`
    CreatedAt time.Time             `json:"created_at,omitempty"`
}
