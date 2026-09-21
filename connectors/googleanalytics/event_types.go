// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package googleanalytics

import (
	"fmt"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/tools/types"
)

var (
	currencyType = types.String().WithMaxBytes(3)
	monetaryType = types.Decimal(20, 2)
	intType      = types.Int(32)
	// genericNumberType should be used to represent the types of those values for
	// which it is not clear what the precision and scale might be. The values
	// chosen here (13 and 3) are considered large enough to represent the values,
	// regardless of what those values mean.
	genericNumberType = types.Decimal(13, 3)
)

type eventType struct {
	ID            string
	Name          string
	DefaultFilter string
	Schema        types.Type // invalid means no schema.
}

var eventTypeByID map[string]*eventType
var eventTypes []*connectors.EventType

// https://developers.google.com/analytics/devguides/collection/protocol/ga4/reference/events.
var measurementProtocolEvents []*eventType

func init() {

	var itemType, promotionItemType types.Type
	var itemTypeProps, creativeItemTypeProps []types.Property
	for _, p := range []types.Property{
		{Name: "item_id", Type: types.String(), DisplayName: "Item ID"},
		{Name: "item_name", Type: types.String(), DisplayName: "Item name"},
		{Name: "affiliation", Type: types.String(), DisplayName: "Affiliation"},
		{Name: "coupon", Type: types.String(), DisplayName: "Coupon code"},
	} {
		itemTypeProps = append(itemTypeProps, p)
		creativeItemTypeProps = append(creativeItemTypeProps, p)
	}
	creativeItemTypeProps = append(creativeItemTypeProps,
		types.Property{Name: "creative_name", Type: types.String(), DisplayName: "Creative name"},
		types.Property{Name: "creative_slot", Type: types.String(), DisplayName: "Creative slot"},
	)
	for _, p := range []types.Property{
		{Name: "discount", Type: monetaryType, DisplayName: "Discount amount"},
		{Name: "index", Type: intType, DisplayName: "Item index"},
		{Name: "item_brand", Type: types.String(), DisplayName: "Item brand"},
		{Name: "item_category", Type: types.String(), DisplayName: "Item category"},
		{Name: "item_category2", Type: types.String(), DisplayName: "Item category 2"},
		{Name: "item_category3", Type: types.String(), DisplayName: "Item category 3"},
		{Name: "item_category4", Type: types.String(), DisplayName: "Item category 4"},
		{Name: "item_category5", Type: types.String(), DisplayName: "Item category 5"},
		{Name: "item_list_id", Type: types.String(), DisplayName: "Item list ID"},
		{Name: "item_list_name", Type: types.String(), DisplayName: "Item list name"},
		{Name: "item_variant", Type: types.String(), DisplayName: "Item variant"},
		{Name: "location_id", Type: types.String(), DisplayName: "Location ID"},
		{Name: "price", Type: monetaryType, DisplayName: "Item price"},
	} {
		itemTypeProps = append(itemTypeProps, p)
		creativeItemTypeProps = append(creativeItemTypeProps, p)
	}
	creativeItemTypeProps = append(creativeItemTypeProps,
		types.Property{Name: "promotion_id", Type: types.String(), DisplayName: "Promotion ID"},
		types.Property{Name: "promotion_name", Type: types.String(), DisplayName: "Promotion name"},
	)
	for _, p := range []types.Property{
		{Name: "quantity", Type: genericNumberType, DisplayName: "Item quantity"},
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
				{Name: "ad_platform", Type: types.String(), DisplayName: "Ad platform"},
				{Name: "ad_source", Type: types.String(), DisplayName: "Ad source"},
				{Name: "ad_format", Type: types.String(), DisplayName: "Ad format"},
				{Name: "ad_unit_name", Type: types.String(), DisplayName: "Ad unit name"},
				{Name: "currency", Type: currencyType, DisplayName: "Currency code"},
				{Name: "value", Type: genericNumberType, DisplayName: "Event value"},
			}),
		},
		{
			ID:            "add_payment_info",
			Name:          "Add Payment Info",
			DefaultFilter: "type is 'track' and event is 'Payment Info Entered'",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, DisplayName: "Currency code"},
				{Name: "value", Type: monetaryType, DisplayName: "Event value"},
				{Name: "coupon", Type: types.String(), DisplayName: "Coupon code"},
				{Name: "payment_type", Type: types.String(), DisplayName: "Payment type"},
				{Name: "items", Type: types.Array(itemType), DisplayName: "Items"},
			}),
		},
		{
			ID:   "add_shipping_info",
			Name: "Add Shipping Info",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, DisplayName: "Currency code"},
				{Name: "value", Type: monetaryType, DisplayName: "Event value"},
				{Name: "coupon", Type: types.String(), DisplayName: "Coupon code"},
				{Name: "shipping_tier", Type: types.String(), DisplayName: "Shipping tier"},
				{Name: "items", Type: types.Array(itemType).WithMinElements(1), CreateRequired: true, DisplayName: "Items"},
			}),
		},
		{
			ID:            "add_to_cart",
			Name:          "Add To Cart",
			DefaultFilter: "type is 'track' and event is 'Product Added'",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, DisplayName: "Currency code"},
				{Name: "value", Type: monetaryType, DisplayName: "Event value"},
				{Name: "items", Type: types.Array(itemType).WithMinElements(1), CreateRequired: true, DisplayName: "Items"},
			}),
		},
		{
			ID:            "add_to_wishlist",
			Name:          "Add To Wishlist",
			DefaultFilter: "type is 'track' and event is 'Product Added to Wishlist'",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, DisplayName: "Currency code"},
				{Name: "value", Type: monetaryType, DisplayName: "Event value"},
				{Name: "items", Type: types.Array(itemType).WithMinElements(1), CreateRequired: true, DisplayName: "Items"},
			}),
		},
		{
			ID:            "begin_checkout",
			Name:          "Begin Checkout",
			DefaultFilter: "type is 'track' and event is 'Checkout Started'",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, DisplayName: "Currency code"},
				{Name: "value", Type: monetaryType, DisplayName: "Event value"},
				{Name: "coupon", Type: types.String(), DisplayName: "Coupon code"},
				{Name: "items", Type: types.Array(itemType).WithMinElements(1), CreateRequired: true, DisplayName: "Items"},
			}),
		},
		{
			ID:   "campaign_details",
			Name: "Campaign Details",
			Schema: types.Object([]types.Property{
				{Name: "campaign_id", Type: types.String(), DisplayName: "Campaign ID"},
				{Name: "campaign", Type: types.String(), DisplayName: "Campaign name"},
				{Name: "source", Type: types.String(), DisplayName: "Traffic source"},
				{Name: "medium", Type: types.String(), DisplayName: "Medium"},
				{Name: "term", Type: types.String(), DisplayName: "Paid search term"},
				{Name: "content", Type: types.String(), DisplayName: "Creative content"},
			}),
		},
		{
			ID:   "close_convert_lead",
			Name: "Close Convert Lead",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, DisplayName: "Currency code"},
				{Name: "value", Type: monetaryType, DisplayName: "Event value"},
			}),
		},
		{
			ID:   "close_unconvert_lead",
			Name: "Close Unconvert Lead",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, DisplayName: "Currency code"},
				{Name: "value", Type: monetaryType, DisplayName: "Event value"},
				{Name: "unconvert_lead_reason", Type: types.String(), DisplayName: "Unconverted lead reason"},
			}),
		},
		{
			ID:   "disqualify_lead",
			Name: "Disqualify Lead",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, DisplayName: "Currency code"},
				{Name: "value", Type: monetaryType, DisplayName: "Event value"},
				{Name: "disqualified_lead_reason", Type: types.String(), DisplayName: "Disqualification reason"},
			}),
		},
		{
			ID:   "earn_virtual_currency",
			Name: "Earn Virtual Currency",
			Schema: types.Object([]types.Property{
				{Name: "virtual_currency_name", Type: types.String(), DisplayName: "Virtual currency name"},
				{Name: "value", Type: genericNumberType, DisplayName: "Event value"},
			}),
		},
		{
			ID:   "generate_lead",
			Name: "Generate Lead",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, DisplayName: "Currency code"},
				{Name: "value", Type: monetaryType, DisplayName: "Event value"},
				{Name: "lead_source", Type: types.String(), DisplayName: "Lead source"},
			}),
		},
		{
			ID:   "join_group",
			Name: "Join Group",
			Schema: types.Object([]types.Property{
				{Name: "group_id", Type: types.String(), DisplayName: "Group ID"},
			}),
		},
		{
			ID:   "level_up",
			Name: "Level Up",
			Schema: types.Object([]types.Property{
				{Name: "level", Type: intType, DisplayName: "Player level"},
				{Name: "character", Type: types.String(), DisplayName: "Player character"},
			}),
		},
		{
			ID:            "login",
			Name:          "Login",
			DefaultFilter: "type is 'track' and event is 'Signed In'",
			Schema: types.Object([]types.Property{
				{Name: "method", Type: types.String(), DisplayName: "Authentication method"},
			}),
		},
		{
			ID:            "page_view",
			Name:          "Page View",
			DefaultFilter: "type is 'page'",
			Schema: types.Object([]types.Property{
				{Name: "page_location", Prefilled: "context.page.url", Type: types.String(), DisplayName: "Page URL"},
				{Name: "page_referrer", Prefilled: "context.page.referrer", Type: types.String(), DisplayName: "Previous page URL"},
				{Name: "page_title", Prefilled: "context.page.title", Type: types.String(), DisplayName: "Page title"},
				{Name: "engagement_time_msec", Prefilled: "1", Type: genericNumberType, DisplayName: "Engagement time", Description: "Measured in milliseconds"},
			}),
		},
		{
			ID:   "post_score",
			Name: "Post Score",
			Schema: types.Object([]types.Property{
				{Name: "score", Type: genericNumberType, CreateRequired: true, DisplayName: "Score value"},
				{Name: "level", Type: intType, DisplayName: "Player level"},
				{Name: "character", Type: types.String(), DisplayName: "Player character"},
			}),
		},
		{
			ID:            "purchase",
			Name:          "Purchase",
			DefaultFilter: "type is 'track' and event is 'Order Completed'",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, DisplayName: "Currency code"},
				{Name: "value", Type: monetaryType, DisplayName: "Event value"},
				{Name: "customer_type", Type: types.String().WithValues("new", "returning"), DisplayName: "Customer type"},
				{Name: "transaction_id", Type: types.String(), CreateRequired: true, DisplayName: "Transaction ID"},
				{Name: "coupon", Type: types.String(), DisplayName: "Coupon code"},
				{Name: "shipping", Type: monetaryType, DisplayName: "Shipping amount"},
				{Name: "tax", Type: monetaryType, DisplayName: "Tax amount"},
				{Name: "items", Type: types.Array(itemType).WithMinElements(1), CreateRequired: true, DisplayName: "Items"},
			}),
		},
		{
			ID:   "qualify_lead",
			Name: "Qualify Lead",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, DisplayName: "Currency code"},
				{Name: "value", Type: monetaryType, DisplayName: "Event value"},
			}),
		},
		{
			ID:            "refund",
			Name:          "Refund",
			DefaultFilter: "type is 'track' and event is 'Order Refunded'",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, DisplayName: "Currency code"},
				{Name: "transaction_id", Type: types.String(), CreateRequired: true, DisplayName: "Transaction ID"},
				{Name: "value", Type: monetaryType, DisplayName: "Event value"},
				{Name: "coupon", Type: types.String(), DisplayName: "Coupon code"},
				{Name: "shipping", Type: monetaryType, DisplayName: "Shipping amount"},
				{Name: "tax", Type: monetaryType, DisplayName: "Tax amount"},
				{Name: "items", Type: types.Array(itemType), DisplayName: "Items"},
			}),
		},
		{
			ID:            "remove_from_cart",
			Name:          "Remove From Cart",
			DefaultFilter: "type is 'track' and event is 'Product Removed'",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, DisplayName: "Currency code"},
				{Name: "value", Type: monetaryType, DisplayName: "Event value"},
				{Name: "items", Type: types.Array(itemType).WithMinElements(1), CreateRequired: true, DisplayName: "Items"},
			}),
		},
		{
			ID:   "screen_view",
			Name: "Screen View",
			Schema: types.Object([]types.Property{
				{Name: "screen_class", Type: types.String(), DisplayName: "Screen class"},
				{Name: "screen_name", Type: types.String(), DisplayName: "Screen name"},
			}),
		},
		{
			ID:            "search",
			Name:          "Search",
			DefaultFilter: "type is 'track' and event is 'Products Searched'",
			Schema: types.Object([]types.Property{
				{Name: "search_term", Type: types.String(), CreateRequired: true, DisplayName: "Search term"},
			}),
		},
		{
			ID:   "select_content",
			Name: "Select Content",
			Schema: types.Object([]types.Property{
				{Name: "content_type", Type: types.String(), DisplayName: "Content type"},
				{Name: "content_id", Type: types.String(), DisplayName: "Content ID"},
			}),
		},
		{
			ID:            "select_item",
			Name:          "Select Item",
			DefaultFilter: "type is 'track' and event is 'Product Clicked'",
			Schema: types.Object([]types.Property{
				{Name: "item_list_id", Type: types.String(), DisplayName: "Item list ID"},
				{Name: "item_list_name", Type: types.String(), DisplayName: "Item list name"},
				{Name: "items", Type: types.Array(itemType).WithMinElements(1), CreateRequired: true, DisplayName: "Items"},
			}),
		},
		{
			ID:            "select_promotion",
			Name:          "Select Promotion",
			DefaultFilter: "type is 'track' and event is 'Promotion Clicked'",
			Schema: types.Object([]types.Property{
				{Name: "creative_name", Type: types.String(), DisplayName: "Creative name"},
				{Name: "creative_slot", Type: types.String(), DisplayName: "Creative slot"},
				{Name: "promotion_id", Type: types.String(), DisplayName: "Promotion ID"},
				{Name: "promotion_name", Type: types.String(), DisplayName: "Promotion name"},
				{Name: "items", Type: types.Array(promotionItemType), DisplayName: "Items"},
			}),
		},
		{
			ID:   "share",
			Name: "Share",
			Schema: types.Object([]types.Property{
				{Name: "method", Type: types.String(), DisplayName: "Sharing method"},
				{Name: "content_type", Type: types.String(), DisplayName: "Content type"},
				{Name: "item_id", Type: types.String(), DisplayName: "Item ID"},
			}),
		},
		{
			ID:            "sign_up",
			Name:          "Sign Up",
			DefaultFilter: "type is 'track' and event is 'Signed Up'",
			Schema: types.Object([]types.Property{
				{Name: "method", Type: types.String(), DisplayName: "Authentication method"},
			}),
		},
		{
			ID:   "spend_virtual_currency",
			Name: "Spend Virtual Currency",
			Schema: types.Object([]types.Property{
				{Name: "value", Type: genericNumberType, CreateRequired: true, DisplayName: "Event value"},
				{Name: "virtual_currency_name", Type: types.String(), CreateRequired: true, DisplayName: "Virtual currency name"},
				{Name: "item_name", Type: types.String(), DisplayName: "Item name"},
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
				{Name: "achievement_id", Type: types.String(), CreateRequired: true, DisplayName: "Achievement ID"},
			}),
		},
		{
			ID:            "view_cart",
			Name:          "View Cart",
			DefaultFilter: "type is 'track' and event is 'Cart Viewed'",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, DisplayName: "Currency code"},
				{Name: "value", Type: monetaryType, DisplayName: "Event value"},
				{Name: "items", Type: types.Array(itemType).WithMinElements(1), CreateRequired: true, DisplayName: "Items"},
			}),
		},
		{
			ID:            "view_item",
			Name:          "View Item",
			DefaultFilter: "type is 'track' and event is 'Product Viewed'",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, DisplayName: "Currency code"},
				{Name: "value", Type: monetaryType, DisplayName: "Event value"},
				{Name: "items", Type: types.Array(itemType).WithMinElements(1), CreateRequired: true, DisplayName: "Items"},
			}),
		},
		{
			ID:            "view_item_list",
			Name:          "View Item List",
			DefaultFilter: "type is 'track' and event is 'Product List Viewed'",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, DisplayName: "Currency code"},
				{Name: "item_list_id", Type: types.String(), DisplayName: "Item list ID"},
				{Name: "item_list_name", Type: types.String(), DisplayName: "Item list name"},
				{Name: "items", Type: types.Array(itemType).WithMinElements(1), CreateRequired: true, DisplayName: "Items"},
			}),
		},
		{
			ID:            "view_promotion",
			Name:          "View Promotion",
			DefaultFilter: "type is 'track' and event is 'Promotion Viewed'",
			Schema: types.Object([]types.Property{
				{Name: "creative_name", Type: types.String(), DisplayName: "Creative name"},
				{Name: "creative_slot", Type: types.String(), DisplayName: "Creative slot"},
				{Name: "promotion_id", Type: types.String(), DisplayName: "Promotion ID"},
				{Name: "promotion_name", Type: types.String(), DisplayName: "Promotion name"},
				{Name: "items", Type: types.Array(promotionItemType).WithMinElements(1), CreateRequired: true, DisplayName: "Items"},
			}),
		},
		{
			ID:   "view_search_results",
			Name: "View Search Results",
			Schema: types.Object([]types.Property{
				{Name: "search_term", Type: types.String(), DisplayName: "Search term"},
			}),
		},
		{
			ID:   "working_lead",
			Name: "Working Lead",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType, DisplayName: "Currency code"},
				{Name: "value", Type: monetaryType, DisplayName: "Event value"},
				{Name: "lead_status", Type: types.String(), DisplayName: "Lead status"},
			}),
		},
	}

	eventTypeByID = make(map[string]*eventType, len(measurementProtocolEvents))
	for _, def := range measurementProtocolEvents {
		eventTypeByID[def.ID] = def
		eventTypes = append(eventTypes, &connectors.EventType{
			ID:            def.ID,
			Name:          def.Name,
			Description:   fmt.Sprintf("Send '%s' events to Google Analytics", def.Name),
			DefaultFilter: def.DefaultFilter,
		})
	}

}
