// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package stripe

import "github.com/krenalis/krenalis/tools/types"

// https://docs.stripe.com/api/customers/object.
//
// Currently, we don't support expanded responses/fields. We have an issue about
// them: https://github.com/krenalis/krenalis/issues/1818.
//
// The "object" and "livemode" fields have been excluded because they are not relevant to Krenalis.
//

var sourceSchema = types.Object([]types.Property{
	{
		Name:        "id",
		Type:        types.String(),
		DisplayName: "Identifier",
	},
	{
		Name:        "name",
		Type:        types.String(),
		Nullable:    true,
		DisplayName: "Name",
	},
	{
		Name:        "email",
		Type:        types.String(),
		Nullable:    true,
		DisplayName: "Account email",
	},
	{
		Name:        "description",
		Type:        types.String(),
		Nullable:    true,
		DisplayName: "Description",
	},
	{
		Name:        "address",
		Type:        sourceAddress,
		Nullable:    true,
		DisplayName: "Billing details",
	},
	{
		Name:        "shipping",
		Type:        sourceShipping,
		Nullable:    true,
		DisplayName: "Shipping details",
	},
	{
		Name:        "phone",
		Type:        types.String(),
		Nullable:    true,
		DisplayName: "Phone",
	},
	{
		Name:        "preferred_locales",
		Type:        types.Array(types.String()),
		Nullable:    true,
		DisplayName: "Locales",
	},
	{
		Name:        "currency",
		Type:        types.String(),
		Nullable:    true,
		DisplayName: "Currency",
	},
	{
		Name:        "invoice_prefix",
		Type:        types.String().WithMaxBytes(12),
		Nullable:    true,
		DisplayName: "Invoice prefix",
	},
	{
		Name:         "next_invoice_sequence",
		Type:         types.Int(32).WithIntRange(1, 1000000000),
		Nullable:     true,
		ReadOptional: true, // if the Stripe account applies account-level sequencing, this parameter is ignored in API requests and excluded from API responses.
		DisplayName:  "Next invoice sequence",
	},
	{
		Name:        "tax_exempt",
		Type:        types.String().WithValues("none", "exempt", "reverse"),
		Nullable:    true,
		DisplayName: "Tax status",
	},
	{
		Name:        "metadata",
		Type:        types.Map(types.String()),
		DisplayName: "Metadata",
	},
	{
		Name:        "balance",
		Type:        types.Int(64).WithIntRange(-1000000000000, 1000000000000),
		DisplayName: "Balance",
	},
	{
		Name:        "created",
		Type:        types.DateTime(),
		DisplayName: "Created",
	},
	{
		Name:        "delinquent",
		Type:        types.Boolean(),
		Nullable:    true,
		DisplayName: "Delinquent",
	},
	{
		Name:        "discount",
		Type:        sourceDiscount,
		Nullable:    true,
		DisplayName: "Active discount",
	},
	{
		Name:        "invoice_settings",
		Type:        sourceInvoiceSettings,
		DisplayName: "Invoice settings",
	},
})

var sourceAddress = types.Object([]types.Property{
	{
		Name:        "country",
		Type:        types.String(), // don't limit to 2 chars: ISO 3166-1 alpha-2 is recommended but not enforced by Stripe.
		Nullable:    true,
		DisplayName: "Country",
	},
	{
		Name:        "line1",
		Type:        types.String(),
		Nullable:    true,
		DisplayName: "Address line 1",
	},
	{
		Name:        "line2",
		Type:        types.String(),
		Nullable:    true,
		DisplayName: "Address line 2",
	},
	{
		Name:        "postal_code",
		Type:        types.String(),
		Nullable:    true,
		DisplayName: "Postal code",
	},
	{
		Name:        "city",
		Type:        types.String(),
		Nullable:    true,
		DisplayName: "City",
	},
	{
		Name:        "state",
		Type:        types.String(),
		Nullable:    true,
		DisplayName: "State/Province",
	},
})

var sourceShipping = types.Object([]types.Property{
	{
		Name:        "name",
		Type:        types.String(),
		DisplayName: "Customer name",
	},
	{
		Name:        "address",
		Type:        sourceAddress,
		DisplayName: "Address",
	},
	{
		Name:        "phone",
		Type:        types.String(),
		Nullable:    true,
		DisplayName: "Phone number",
	},
})

var sourceDiscount = types.Object([]types.Property{
	{
		Name:        "id",
		Type:        types.String(),
		DisplayName: "Discount ID",
	},
	{
		Name:        "checkout_session",
		Type:        types.String(),
		Nullable:    true,
		DisplayName: "Checkout session for coupon",
	},
	{
		Name:        "coupon",
		Type:        sourceCoupon,
		DisplayName: "Coupon applied",
	},
	{
		Name:        "end",
		Type:        types.DateTime(),
		Nullable:    true,
		DisplayName: "End date",
	},
	{
		Name:        "invoice",
		Type:        types.String(),
		Nullable:    true,
		DisplayName: "Applied invoice",
	},
	{
		Name:        "invoice_item",
		Type:        types.String(),
		Nullable:    true,
		DisplayName: "Invoice item ID",
	},
	{
		Name:        "start",
		Type:        types.DateTime(),
		DisplayName: "Date coupon applied",
	},
	{
		Name:        "subscription",
		Type:        types.String(),
		Nullable:    true,
		DisplayName: "Subscription ID",
	},
	{
		Name:        "subscription_item",
		Type:        types.String(),
		Nullable:    true,
		DisplayName: "Subscription item ID",
	},
})

var sourceCoupon = types.Object([]types.Property{
	{
		Name:        "name",
		Type:        types.String(),
		DisplayName: "Name",
	},
	{
		Name:        "id",
		Type:        types.String(),
		DisplayName: "ID",
	},
	{
		Name:        "percent_off",
		Type:        types.Float(64),
		DisplayName: "Percent off",
	},
	{
		Name:        "amount_off",
		Type:        types.Int(64),
		Nullable:    true,
		DisplayName: "Amount off",
	},
	{
		Name:        "duration",
		Type:        types.String().WithValues("forever", "once", "repeating"),
		DisplayName: "Duration",
	},
	{
		Name:        "redeem_by",
		Type:        types.DateTime(),
		Nullable:    true,
		DisplayName: "Redeem by date",
	},
	{
		Name:        "max_redemptions",
		Type:        types.Int(32),
		Nullable:    true,
		DisplayName: "Maximum redemptions",
	},
	{
		Name:        "times_redeemed",
		Type:        types.Int(32),
		DisplayName: "Times redeemed",
	},
	{
		Name:        "created",
		Type:        types.DateTime(),
		DisplayName: "Created",
	},
	{
		Name:        "currency",
		Type:        types.String(),
		DisplayName: "Currency",
	},
	{
		Name:        "duration_in_months",
		Type:        types.Int(32),
		Nullable:    true,
		DisplayName: "Duration in months",
	},
	{
		Name:        "metadata",
		Type:        types.Map(types.String()),
		DisplayName: "Metadata",
	},
	{
		Name:        "valid",
		Type:        types.Boolean(),
		DisplayName: "Still valid",
	},
})

var sourceInvoiceSettings = types.Object([]types.Property{
	{
		Name: "rendering_options",
		Type: types.Object([]types.Property{
			{
				Name:        "amount_tax_display",
				Type:        types.String(),
				Nullable:    true,
				DisplayName: "Amount tax display",
			},
			{
				Name:        "template",
				Type:        types.String(),
				Nullable:    true,
				DisplayName: "Template",
			},
		}),
		Nullable:    true,
		DisplayName: "Rendering options",
	},
	{
		Name:        "footer",
		Type:        types.String(),
		Nullable:    true,
		DisplayName: "Footer",
	},
	{
		Name: "custom_fields",
		Type: types.Array(types.Object([]types.Property{
			{
				Name:        "name",
				Type:        types.String().WithMaxLength(40),
				DisplayName: "Field name",
			},
			{
				Name:        "value",
				Type:        types.String().WithMaxBytes(140),
				DisplayName: "Field value",
			},
		})),
		Nullable:    true,
		DisplayName: "Custom fields",
	},
})
