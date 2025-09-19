//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package googleanalytics

import (
	"fmt"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/types"
)

var (
	currencyType = types.Text().WithByteLen(3)
	monetaryType = types.Decimal(20, 2)
	intType      = types.Int(32)
	// genericNumberType should be used to represent the types of those values for
	// which it is not clear what the precision and scale might be. The values
	// chosen here (13 and 3) are considered large enough to represent the values,
	// regardless of what those values mean.
	genericNumberType = types.Decimal(13, 3)
)

type eventType struct {
	ID     string
	Name   string
	Schema types.Type // invalid means no schema.
}

var eventTypeByID map[string]*eventType
var meergoEventTypes []*meergo.EventType

// https://developers.google.com/analytics/devguides/collection/protocol/ga4/reference/events.
var measurementProtocolEvents []*eventType

func init() {

	var itemType, promotionItemType types.Type
	var itemTypeProps, creativeItemTypeProps []types.Property
	for _, p := range []types.Property{
		{Name: "item_id", Type: types.Text(), Description: "Item ID"},
		{Name: "item_name", Type: types.Text(), Description: "Item name"},
		{Name: "affiliation", Type: types.Text(), Description: "Affiliation"},
		{Name: "coupon", Type: types.Text(), Description: "Coupon code"},
	} {
		itemTypeProps = append(itemTypeProps, p)
		creativeItemTypeProps = append(creativeItemTypeProps, p)
	}
	creativeItemTypeProps = append(creativeItemTypeProps,
		types.Property{Name: "creative_name", Type: types.Text(), Description: "Creative name"},
		types.Property{Name: "creative_slot", Type: types.Text(), Description: "Creative slot"},
	)
	for _, p := range []types.Property{
		{Name: "discount", Type: monetaryType, Description: "Discount amount"},
		{Name: "index", Type: intType, Description: "Item index"},
		{Name: "item_brand", Type: types.Text(), Description: "Item brand"},
		{Name: "item_category", Type: types.Text(), Description: "Item category"},
		{Name: "item_category2", Type: types.Text(), Description: "Item category 2"},
		{Name: "item_category3", Type: types.Text(), Description: "Item category 3"},
		{Name: "item_category4", Type: types.Text(), Description: "Item category 4"},
		{Name: "item_category5", Type: types.Text(), Description: "Item category 5"},
		{Name: "item_list_id", Type: types.Text(), Description: "Item list ID"},
		{Name: "item_list_name", Type: types.Text(), Description: "Item list name"},
		{Name: "item_variant", Type: types.Text(), Description: "Item variant"},
		{Name: "location_id", Type: types.Text(), Description: "Location ID"},
		{Name: "price", Type: monetaryType, Description: "Item price"},
	} {
		itemTypeProps = append(itemTypeProps, p)
		creativeItemTypeProps = append(creativeItemTypeProps, p)
	}
	creativeItemTypeProps = append(creativeItemTypeProps,
		types.Property{Name: "promotion_id", Type: types.Text(), Description: "Promotion ID"},
		types.Property{Name: "promotion_name", Type: types.Text(), Description: "Promotion name"},
	)
	for _, p := range []types.Property{
		{Name: "quantity", Type: genericNumberType, Description: "Item quantity"},
	} {
		itemTypeProps = append(itemTypeProps, p)
		creativeItemTypeProps = append(creativeItemTypeProps, p)
	}
	itemType = types.Object(itemTypeProps)
	promotionItemType = types.Object(creativeItemTypeProps)

	measurementProtocolEvents = []*eventType{
		{
			ID:   "ad_impression",
			Name: "Ad Impression",
			Schema: types.Object([]types.Property{
				{Name: "ad_platform", Type: types.Text(), Description: "Ad platform"},
				{Name: "ad_source", Type: types.Text(), Description: "Ad source"},
				{Name: "ad_format", Type: types.Text(), Description: "Ad format"},
				{Name: "ad_unit_name", Type: types.Text(), Description: "Ad unit name"},
				{Name: "currency", Type: currencyType, Description: "Currency code"},
				{Name: "value", Type: genericNumberType, Description: "Event value"},
			}),
		},
		{
			ID:   "add_payment_info",
			Name: "Add Payment Info",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, Description: "Currency code"},
				{Name: "value", Type: monetaryType, Description: "Event value"},
				{Name: "coupon", Type: types.Text(), Description: "Coupon code"},
				{Name: "payment_type", Type: types.Text(), Description: "Payment type"},
				{Name: "items", Type: types.Array(itemType), Description: "Items"},
			}),
		},
		{
			ID:   "add_shipping_info",
			Name: "Add Shipping Info",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, Description: "Currency code"},
				{Name: "value", Type: monetaryType, Description: "Event value"},
				{Name: "coupon", Type: types.Text(), Description: "Coupon code"},
				{Name: "shipping_tier", Type: types.Text(), Description: "Shipping tier"},
				{Name: "items", Type: types.Array(itemType).WithMinElements(1), CreateRequired: true, Description: "Items"},
			}),
		},
		{
			ID:   "add_to_cart",
			Name: "Add To Cart",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, Description: "Currency code"},
				{Name: "value", Type: monetaryType, Description: "Event value"},
				{Name: "items", Type: types.Array(itemType).WithMinElements(1), CreateRequired: true, Description: "Items"},
			}),
		},
		{
			ID:   "add_to_wishlist",
			Name: "Add To Wishlist",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, Description: "Currency code"},
				{Name: "value", Type: monetaryType, Description: "Event value"},
				{Name: "items", Type: types.Array(itemType).WithMinElements(1), CreateRequired: true, Description: "Items"},
			}),
		},
		{
			ID:   "begin_checkout",
			Name: "Begin Checkout",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, Description: "Currency code"},
				{Name: "value", Type: monetaryType, Description: "Event value"},
				{Name: "coupon", Type: types.Text(), Description: "Coupon code"},
				{Name: "items", Type: types.Array(itemType).WithMinElements(1), CreateRequired: true, Description: "Items"},
			}),
		},
		{
			ID:   "campaign_details",
			Name: "Campaign Details",
			Schema: types.Object([]types.Property{
				{Name: "campaign_id", Type: types.Text(), Description: "Campaign ID"},
				{Name: "campaign", Type: types.Text(), Description: "Campaign name"},
				{Name: "source", Type: types.Text(), Description: "Traffic source"},
				{Name: "medium", Type: types.Text(), Description: "Medium"},
				{Name: "term", Type: types.Text(), Description: "Paid search term"},
				{Name: "content", Type: types.Text(), Description: "Creative content"},
			}),
		},
		{
			ID:   "close_convert_lead",
			Name: "Close Convert Lead",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, Description: "Currency code"},
				{Name: "value", Type: monetaryType, Description: "Event value"},
			}),
		},
		{
			ID:   "close_unconvert_lead",
			Name: "Close Unconvert Lead",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, Description: "Currency code"},
				{Name: "value", Type: monetaryType, Description: "Event value"},
				{Name: "unconvert_lead_reason", Type: types.Text(), Description: "Unconverted lead reason"},
			}),
		},
		{
			ID:   "disqualify_lead",
			Name: "Disqualify Lead",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, Description: "Currency code"},
				{Name: "value", Type: monetaryType, Description: "Event value"},
				{Name: "disqualified_lead_reason", Type: types.Text(), Description: "Disqualification reason"},
			}),
		},
		{
			ID:   "earn_virtual_currency",
			Name: "Earn Virtual Currency",
			Schema: types.Object([]types.Property{
				{Name: "virtual_currency_name", Type: types.Text(), Description: "Virtual currency name"},
				{Name: "value", Type: genericNumberType, Description: "Event value"},
			}),
		},
		{
			ID:   "generate_lead",
			Name: "Generate Lead",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, Description: "Currency code"},
				{Name: "value", Type: monetaryType, Description: "Event value"},
				{Name: "lead_source", Type: types.Text(), Description: "Lead source"},
			}),
		},
		{
			ID:   "join_group",
			Name: "Join Group",
			Schema: types.Object([]types.Property{
				{Name: "group_id", Type: types.Text(), Description: "Group ID"},
			}),
		},
		{
			ID:   "level_up",
			Name: "Level Up",
			Schema: types.Object([]types.Property{
				{Name: "level", Type: intType, Description: "Player level"},
				{Name: "character", Type: types.Text(), Description: "Player character"},
			}),
		},
		{
			ID:   "login",
			Name: "Login",
			Schema: types.Object([]types.Property{
				{Name: "method", Type: types.Text(), Description: "Authentication method"},
			}),
		},
		{
			ID:   "post_score",
			Name: "Post Score",
			Schema: types.Object([]types.Property{
				{Name: "score", Type: genericNumberType, CreateRequired: true, Description: "Score value"},
				{Name: "level", Type: intType, Description: "Player level"},
				{Name: "character", Type: types.Text(), Description: "Player character"},
			}),
		},
		{
			ID:   "purchase",
			Name: "Purchase",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, Description: "Currency code"},
				{Name: "value", Type: monetaryType, Description: "Event value"},
				{Name: "customer_type", Type: types.Text().WithValues("new", "returning"), Description: "Customer type"},
				{Name: "transaction_id", Type: types.Text(), CreateRequired: true, Description: "Transaction ID"},
				{Name: "coupon", Type: types.Text(), Description: "Coupon code"},
				{Name: "shipping", Type: monetaryType, Description: "Shipping amount"},
				{Name: "tax", Type: monetaryType, Description: "Tax amount"},
				{Name: "items", Type: types.Array(itemType).WithMinElements(1), CreateRequired: true, Description: "Items"},
			}),
		},
		{
			ID:   "qualify_lead",
			Name: "Qualify Lead",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, Description: "Currency code"},
				{Name: "value", Type: monetaryType, Description: "Event value"},
			}),
		},
		{
			ID:   "refund",
			Name: "Refund",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, Description: "Currency code"},
				{Name: "transaction_id", Type: types.Text(), CreateRequired: true, Description: "Transaction ID"},
				{Name: "value", Type: monetaryType, Description: "Event value"},
				{Name: "coupon", Type: types.Text(), Description: "Coupon code"},
				{Name: "shipping", Type: monetaryType, Description: "Shipping amount"},
				{Name: "tax", Type: monetaryType, Description: "Tax amount"},
				{Name: "items", Type: types.Array(itemType), Description: "Items"},
			}),
		},
		{
			ID:   "remove_from_cart",
			Name: "Remove From Cart",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, Description: "Currency code"},
				{Name: "value", Type: monetaryType, Description: "Event value"},
				{Name: "items", Type: types.Array(itemType).WithMinElements(1), CreateRequired: true, Description: "Items"},
			}),
		},
		{
			ID:   "screen_view",
			Name: "Screen View",
			Schema: types.Object([]types.Property{
				{Name: "screen_class", Type: types.Text(), Description: "Screen class"},
				{Name: "screen_name", Type: types.Text(), Description: "Screen name"},
			}),
		},
		{
			ID:   "search",
			Name: "Search",
			Schema: types.Object([]types.Property{
				{Name: "search_term", Type: types.Text(), CreateRequired: true, Description: "Search term"},
			}),
		},
		{
			ID:   "select_content",
			Name: "Select Content",
			Schema: types.Object([]types.Property{
				{Name: "content_type", Type: types.Text(), Description: "Content type"},
				{Name: "content_id", Type: types.Text(), Description: "Content ID"},
			}),
		},
		{
			ID:   "select_item",
			Name: "Select Item",
			Schema: types.Object([]types.Property{
				{Name: "item_list_id", Type: types.Text(), Description: "Item list ID"},
				{Name: "item_list_name", Type: types.Text(), Description: "Item list name"},
				{Name: "items", Type: types.Array(itemType).WithMinElements(1), CreateRequired: true, Description: "Items"},
			}),
		},
		{
			ID:   "select_promotion",
			Name: "Select Promotion",
			Schema: types.Object([]types.Property{
				{Name: "creative_name", Type: types.Text(), Description: "Creative name"},
				{Name: "creative_slot", Type: types.Text(), Description: "Creative slot"},
				{Name: "promotion_id", Type: types.Text(), Description: "Promotion ID"},
				{Name: "promotion_name", Type: types.Text(), Description: "Promotion name"},
				{Name: "items", Type: types.Array(promotionItemType), Description: "Items"},
			}),
		},
		{
			ID:   "share",
			Name: "Share",
			Schema: types.Object([]types.Property{
				{Name: "method", Type: types.Text(), Description: "Sharing method"},
				{Name: "content_type", Type: types.Text(), Description: "Content type"},
				{Name: "item_id", Type: types.Text(), Description: "Item ID"},
			}),
		},
		{
			ID:   "sign_up",
			Name: "Sign Up",
			Schema: types.Object([]types.Property{
				{Name: "method", Type: types.Text(), Description: "Authentication method"},
			}),
		},
		{
			ID:   "spend_virtual_currency",
			Name: "Spend Virtual Currency",
			Schema: types.Object([]types.Property{
				{Name: "value", Type: genericNumberType, CreateRequired: true, Description: "Event value"},
				{Name: "virtual_currency_name", Type: types.Text(), CreateRequired: true, Description: "Virtual currency name"},
				{Name: "item_name", Type: types.Text(), Description: "Item name"},
			}),
		},
		{
			ID:     "tutorial_begin",
			Name:   "Tutorial Begin",
			Schema: types.Type{},
		},
		{
			ID:     "tutorial_complete",
			Name:   "Tutorial Complete",
			Schema: types.Type{},
		},
		{
			ID:   "unlock_achievement",
			Name: "Unlock Achievement",
			Schema: types.Object([]types.Property{
				{Name: "achievement_id", Type: types.Text(), CreateRequired: true, Description: "Achievement ID"},
			}),
		},
		{
			ID:   "view_cart",
			Name: "View Cart",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, Description: "Currency code"},
				{Name: "value", Type: monetaryType, Description: "Event value"},
				{Name: "items", Type: types.Array(itemType).WithMinElements(1), CreateRequired: true, Description: "Items"},
			}),
		},
		{
			ID:   "view_item",
			Name: "View Item",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, Description: "Currency code"},
				{Name: "value", Type: monetaryType, Description: "Event value"},
				{Name: "items", Type: types.Array(itemType).WithMinElements(1), CreateRequired: true, Description: "Items"},
			}),
		},
		{
			ID:   "view_item_list",
			Name: "View Item List",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, Description: "Currency code"},
				{Name: "item_list_id", Type: types.Text(), Description: "Item list ID"},
				{Name: "item_list_name", Type: types.Text(), Description: "Item list name"},
				{Name: "items", Type: types.Array(itemType).WithMinElements(1), CreateRequired: true, Description: "Items"},
			}),
		},
		{
			ID:   "view_promotion",
			Name: "View Promotion",
			Schema: types.Object([]types.Property{
				{Name: "creative_name", Type: types.Text(), Description: "Creative name"},
				{Name: "creative_slot", Type: types.Text(), Description: "Creative slot"},
				{Name: "promotion_id", Type: types.Text(), Description: "Promotion ID"},
				{Name: "promotion_name", Type: types.Text(), Description: "Promotion name"},
				{Name: "items", Type: types.Array(promotionItemType).WithMinElements(1), CreateRequired: true, Description: "Items"},
			}),
		},
		{
			ID:   "view_search_results",
			Name: "View Search Results",
			Schema: types.Object([]types.Property{
				{Name: "search_term", Type: types.Text(), Description: "Search term"},
			}),
		},
		{
			ID:   "working_lead",
			Name: "Working Lead",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, Description: "Currency code"},
				{Name: "value", Type: monetaryType, Description: "Event value"},
				{Name: "lead_status", Type: types.Text(), Description: "Lead status"},
			}),
		},
	}

	eventTypeByID = make(map[string]*eventType, len(measurementProtocolEvents))
	for _, def := range measurementProtocolEvents {
		eventTypeByID[def.ID] = def
		meergoEventTypes = append(meergoEventTypes, &meergo.EventType{
			ID:          def.ID,
			Name:        def.Name,
			Description: fmt.Sprintf("Send '%s' events to Google Analytics", def.Name),
		})
	}

}
