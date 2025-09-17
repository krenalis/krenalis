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
		{Name: "item_id", Type: types.Text()},
		{Name: "item_name", Type: types.Text()},
		{Name: "affiliation", Type: types.Text()},
		{Name: "coupon", Type: types.Text()},
	} {
		itemTypeProps = append(itemTypeProps, p)
		creativeItemTypeProps = append(creativeItemTypeProps, p)
	}
	creativeItemTypeProps = append(creativeItemTypeProps,
		types.Property{Name: "creative_name", Type: types.Text()},
		types.Property{Name: "creative_slot", Type: types.Text()},
	)
	for _, p := range []types.Property{
		{Name: "discount", Type: monetaryType},
		{Name: "index", Type: intType},
		{Name: "item_brand", Type: types.Text()},
		{Name: "item_category", Type: types.Text()},
		{Name: "item_category2", Type: types.Text()},
		{Name: "item_category3", Type: types.Text()},
		{Name: "item_category4", Type: types.Text()},
		{Name: "item_category5", Type: types.Text()},
		{Name: "item_list_id", Type: types.Text()},
		{Name: "item_list_name", Type: types.Text()},
		{Name: "item_variant", Type: types.Text()},
		{Name: "location_id", Type: types.Text()},
		{Name: "price", Type: monetaryType},
	} {
		itemTypeProps = append(itemTypeProps, p)
		creativeItemTypeProps = append(creativeItemTypeProps, p)
	}
	creativeItemTypeProps = append(creativeItemTypeProps,
		types.Property{Name: "promotion_id", Type: types.Text()},
		types.Property{Name: "promotion_name", Type: types.Text()},
	)
	for _, p := range []types.Property{
		{Name: "quantity", Type: genericNumberType},
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
				{Name: "ad_platform", Type: types.Text()},
				{Name: "ad_source", Type: types.Text()},
				{Name: "ad_format", Type: types.Text()},
				{Name: "ad_unit_name", Type: types.Text()},
				{Name: "currency", Type: currencyType},
				{Name: "value", Type: genericNumberType},
			}),
		},
		{
			ID:   "add_payment_info",
			Name: "Add Payment Info",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType},
				{Name: "value", Type: monetaryType},
				{Name: "coupon", Type: types.Text()},
				{Name: "payment_type", Type: types.Text()},
				{Name: "items", Type: types.Array(itemType)},
			}),
		},
		{
			ID:   "add_shipping_info",
			Name: "Add Shipping Info",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType},
				{Name: "value", Type: monetaryType},
				{Name: "coupon", Type: types.Text()},
				{Name: "shipping_tier", Type: types.Text()},
				{Name: "items", Type: types.Array(itemType).WithMinElements(1), CreateRequired: true},
			}),
		},
		{
			ID:   "add_to_cart",
			Name: "Add To Cart",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType},
				{Name: "value", Type: monetaryType},
				{Name: "items", Type: types.Array(itemType).WithMinElements(1), CreateRequired: true},
			}),
		},
		{
			ID:   "add_to_wishlist",
			Name: "Add To Wishlist",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType},
				{Name: "value", Type: monetaryType},
				{Name: "items", Type: types.Array(itemType).WithMinElements(1), CreateRequired: true},
			}),
		},
		{
			ID:   "begin_checkout",
			Name: "Begin Checkout",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType},
				{Name: "value", Type: monetaryType},
				{Name: "coupon", Type: types.Text()},
				{Name: "items", Type: types.Array(itemType).WithMinElements(1), CreateRequired: true},
			}),
		},
		{
			ID:   "campaign_details",
			Name: "Campaign Details",
			Schema: types.Object([]types.Property{
				{Name: "campaign_id", Type: types.Text()},
				{Name: "campaign", Type: types.Text()},
				{Name: "source", Type: types.Text()},
				{Name: "medium", Type: types.Text()},
				{Name: "term", Type: types.Text()},
				{Name: "content", Type: types.Text()},
			}),
		},
		{
			ID:   "close_convert_lead",
			Name: "Close Convert Lead",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType},
				{Name: "value", Type: monetaryType},
			}),
		},
		{
			ID:   "close_unconvert_lead",
			Name: "Close Unconvert Lead",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType},
				{Name: "value", Type: monetaryType},
				{Name: "unconvert_lead_reason", Type: types.Text()},
			}),
		},
		{
			ID:   "disqualify_lead",
			Name: "Disqualify Lead",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType},
				{Name: "value", Type: monetaryType},
				{Name: "disqualified_lead_reason", Type: types.Text()},
			}),
		},
		{
			ID:   "earn_virtual_currency",
			Name: "Earn Virtual Currency",
			Schema: types.Object([]types.Property{
				{Name: "virtual_currency_name", Type: types.Text()},
				{Name: "value", Type: genericNumberType},
			}),
		},
		{
			ID:   "generate_lead",
			Name: "Generate Lead",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType},
				{Name: "value", Type: monetaryType},
				{Name: "lead_source", Type: types.Text()},
			}),
		},
		{
			ID:   "join_group",
			Name: "Join Group",
			Schema: types.Object([]types.Property{
				{Name: "group_id", Type: types.Text()},
			}),
		},
		{
			ID:   "level_up",
			Name: "Level Up",
			Schema: types.Object([]types.Property{
				{Name: "level", Type: intType},
				{Name: "character", Type: types.Text()},
			}),
		},
		{
			ID:   "login",
			Name: "Login",
			Schema: types.Object([]types.Property{
				{Name: "method", Type: types.Text()},
			}),
		},
		{
			ID:   "post_score",
			Name: "Post Score",
			Schema: types.Object([]types.Property{
				{Name: "score", Type: genericNumberType, CreateRequired: true},
				{Name: "level", Type: intType},
				{Name: "character", Type: types.Text()},
			}),
		},
		{
			ID:   "purchase",
			Name: "Purchase",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType},
				{Name: "value", Type: monetaryType},
				{Name: "customer_type", Type: types.Text().WithValues("new", "returning")},
				{Name: "transaction_id", Type: types.Text(), CreateRequired: true},
				{Name: "coupon", Type: types.Text()},
				{Name: "shipping", Type: monetaryType},
				{Name: "tax", Type: monetaryType},
				{Name: "items", Type: types.Array(itemType).WithMinElements(1), CreateRequired: true},
			}),
		},
		{
			ID:   "qualify_lead",
			Name: "Qualify Lead",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType},
				{Name: "value", Type: monetaryType},
			}),
		},
		{
			ID:   "refund",
			Name: "Refund",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType},
				{Name: "transaction_id", Type: types.Text(), CreateRequired: true},
				{Name: "value", Type: monetaryType},
				{Name: "coupon", Type: types.Text()},
				{Name: "shipping", Type: monetaryType},
				{Name: "tax", Type: monetaryType},
				{Name: "items", Type: types.Array(itemType)},
			}),
		},
		{
			ID:   "remove_from_cart",
			Name: "Remove From Cart",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType},
				{Name: "value", Type: monetaryType},
				{Name: "items", Type: types.Array(itemType).WithMinElements(1), CreateRequired: true},
			}),
		},
		{
			ID:   "screen_view",
			Name: "Screen View",
			Schema: types.Object([]types.Property{
				{Name: "screen_class", Type: types.Text()},
				{Name: "screen_name", Type: types.Text()},
			}),
		},
		{
			ID:   "search",
			Name: "Search",
			Schema: types.Object([]types.Property{
				{Name: "search_term", Type: types.Text(), CreateRequired: true},
			}),
		},
		{
			ID:   "select_content",
			Name: "Select Content",
			Schema: types.Object([]types.Property{
				{Name: "content_type", Type: types.Text()},
				{Name: "content_id", Type: types.Text()},
			}),
		},
		{
			ID:   "select_item",
			Name: "Select Item",
			Schema: types.Object([]types.Property{
				{Name: "item_list_id", Type: types.Text()},
				{Name: "item_list_name", Type: types.Text()},
				{Name: "items", Type: types.Array(itemType).WithMinElements(1), CreateRequired: true},
			}),
		},
		{
			ID:   "select_promotion",
			Name: "Select Promotion",
			Schema: types.Object([]types.Property{
				{Name: "creative_name", Type: types.Text()},
				{Name: "creative_slot", Type: types.Text()},
				{Name: "promotion_id", Type: types.Text()},
				{Name: "promotion_name", Type: types.Text()},
				{Name: "items", Type: types.Array(promotionItemType)},
			}),
		},
		{
			ID:   "share",
			Name: "Share",
			Schema: types.Object([]types.Property{
				{Name: "method", Type: types.Text()},
				{Name: "content_type", Type: types.Text()},
				{Name: "item_id", Type: types.Text()},
			}),
		},
		{
			ID:   "sign_up",
			Name: "Sign Up",
			Schema: types.Object([]types.Property{
				{Name: "method", Type: types.Text()},
			}),
		},
		{
			ID:   "spend_virtual_currency",
			Name: "Spend Virtual Currency",
			Schema: types.Object([]types.Property{
				{Name: "value", Type: genericNumberType, CreateRequired: true},
				{Name: "virtual_currency_name", Type: types.Text(), CreateRequired: true},
				{Name: "item_name", Type: types.Text()},
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
				{Name: "achievement_id", Type: types.Text(), CreateRequired: true},
			}),
		},
		{
			ID:   "view_cart",
			Name: "View Cart",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType},
				{Name: "value", Type: monetaryType},
				{Name: "items", Type: types.Array(itemType).WithMinElements(1), CreateRequired: true},
			}),
		},
		{
			ID:   "view_item",
			Name: "View Item",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType},
				{Name: "value", Type: monetaryType},
				{Name: "items", Type: types.Array(itemType).WithMinElements(1), CreateRequired: true},
			}),
		},
		{
			ID:   "view_item_list",
			Name: "View Item List",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType},
				{Name: "item_list_id", Type: types.Text()},
				{Name: "item_list_name", Type: types.Text()},
				{Name: "items", Type: types.Array(itemType).WithMinElements(1), CreateRequired: true},
			}),
		},
		{
			ID:   "view_promotion",
			Name: "View Promotion",
			Schema: types.Object([]types.Property{
				{Name: "creative_name", Type: types.Text()},
				{Name: "creative_slot", Type: types.Text()},
				{Name: "promotion_id", Type: types.Text()},
				{Name: "promotion_name", Type: types.Text()},
				{Name: "items", Type: types.Array(promotionItemType).WithMinElements(1), CreateRequired: true},
			}),
		},
		{
			ID:   "view_search_results",
			Name: "View Search Results",
			Schema: types.Object([]types.Property{
				{Name: "search_term", Type: types.Text()},
			}),
		},
		{
			ID:   "working_lead",
			Name: "Working Lead",
			Schema: types.Object([]types.Property{
				{Name: "currency", Type: currencyType},
				{Name: "value", Type: monetaryType},
				{Name: "lead_status", Type: types.Text()},
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
