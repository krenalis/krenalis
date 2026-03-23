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
// The "object" and "livemode" fields have been excluded because they are not relevant to Meergo.
//

var sourceSchema = types.Object([]types.Property{
	{
		Name:        "id",
		Type:        types.String(),
		Description: "Identifier",
	},
	{
		Name:        "name",
		Type:        types.String(),
		Nullable:    true,
		Description: "Name",
	},
	{
		Name:        "email",
		Type:        types.String(),
		Nullable:    true,
		Description: "Account email",
	},
	{
		Name:        "description",
		Type:        types.String(),
		Nullable:    true,
		Description: "Description",
	},
	{
		Name:        "address",
		Type:        sourceAddress,
		Nullable:    true,
		Description: "Billing details",
	},
	{
		Name:        "shipping",
		Type:        sourceShipping,
		Nullable:    true,
		Description: "Shipping details",
	},
	{
		Name:        "phone",
		Type:        types.String(),
		Nullable:    true,
		Description: "Phone",
	},
	{
		Name:        "preferred_locales",
		Type:        types.Array(types.String()),
		Nullable:    true,
		Description: "Locales",
	},
	{
		Name:        "currency",
		Type:        types.String(),
		Nullable:    true,
		Description: "Currency",
	},
	{
		Name:        "invoice_prefix",
		Type:        types.String().WithMaxBytes(12),
		Nullable:    true,
		Description: "Invoice prefix",
	},
	{
		Name:         "next_invoice_sequence",
		Type:         types.Int(32).WithIntRange(1, 1000000000),
		Nullable:     true,
		ReadOptional: true, // if the Stripe account applies account-level sequencing, this parameter is ignored in API requests and excluded from API responses.
		Description:  "Next invoice sequence",
	},
	{
		Name:        "tax_exempt",
		Type:        types.String().WithValues("none", "exempt", "reverse"),
		Nullable:    true,
		Description: "Tax status",
	},
	{
		Name:        "metadata",
		Type:        types.Map(types.String()),
		Description: "Metadata",
	},
	{
		Name:        "balance",
		Type:        types.Int(64).WithIntRange(-1000000000000, 1000000000000),
		Description: "Balance",
	},
	{
		Name:        "created",
		Type:        types.DateTime(),
		Description: "Created",
	},
	{
		Name:        "delinquent",
		Type:        types.Boolean(),
		Nullable:    true,
		Description: "Delinquent",
	},
	{
		Name:        "discount",
		Type:        sourceDiscount,
		Nullable:    true,
		Description: "Active discount",
	},
	{
		Name:        "invoice_settings",
		Type:        sourceInvoiceSettings,
		Description: "Invoice settings",
	},
})

var sourceAddress = types.Object([]types.Property{
	{
		Name:        "country",
		Type:        types.String(), // don't limit to 2 chars: ISO 3166-1 alpha-2 is recommended but not enforced by Stripe.
		Nullable:    true,
		Description: "Country",
	},
	{
		Name:        "line1",
		Type:        types.String(),
		Nullable:    true,
		Description: "Address line 1",
	},
	{
		Name:        "line2",
		Type:        types.String(),
		Nullable:    true,
		Description: "Address line 2",
	},
	{
		Name:        "postal_code",
		Type:        types.String(),
		Nullable:    true,
		Description: "Postal code",
	},
	{
		Name:        "city",
		Type:        types.String(),
		Nullable:    true,
		Description: "City",
	},
	{
		Name:        "state",
		Type:        types.String(),
		Nullable:    true,
		Description: "State/Province",
	},
})

var sourceShipping = types.Object([]types.Property{
	{
		Name:        "name",
		Type:        types.String(),
		Description: "Customer name",
	},
	{
		Name:        "address",
		Type:        sourceAddress,
		Description: "Address",
	},
	{
		Name:        "phone",
		Type:        types.String(),
		Nullable:    true,
		Description: "Phone number",
	},
})

var sourceDiscount = types.Object([]types.Property{
	{
		Name:        "id",
		Type:        types.String(),
		Description: "Discount ID",
	},
	{
		Name:        "checkout_session",
		Type:        types.String(),
		Nullable:    true,
		Description: "Checkout session for coupon",
	},
	{
		Name:        "coupon",
		Type:        sourceCoupon,
		Description: "Coupon applied",
	},
	{
		Name:        "end",
		Type:        types.DateTime(),
		Nullable:    true,
		Description: "End date",
	},
	{
		Name:        "invoice",
		Type:        types.String(),
		Nullable:    true,
		Description: "Applied invoice",
	},
	{
		Name:        "invoice_item",
		Type:        types.String(),
		Nullable:    true,
		Description: "Invoice item ID",
	},
	{
		Name:        "start",
		Type:        types.DateTime(),
		Description: "Date coupon applied",
	},
	{
		Name:        "subscription",
		Type:        types.String(),
		Nullable:    true,
		Description: "Subscription ID",
	},
	{
		Name:        "subscription_item",
		Type:        types.String(),
		Nullable:    true,
		Description: "Subscription item ID",
	},
})

var sourceCoupon = types.Object([]types.Property{
	{
		Name:        "name",
		Type:        types.String(),
		Description: "Name",
	},
	{
		Name:        "id",
		Type:        types.String(),
		Description: "ID",
	},
	{
		Name:        "percent_off",
		Type:        types.Float(64),
		Description: "Percent off",
	},
	{
		Name:        "amount_off",
		Type:        types.Int(64),
		Nullable:    true,
		Description: "Amount off",
	},
	{
		Name:        "duration",
		Type:        types.String().WithValues("forever", "once", "repeating"),
		Description: "Duration",
	},
	{
		Name:        "redeem_by",
		Type:        types.DateTime(),
		Nullable:    true,
		Description: "Redeem by date",
	},
	{
		Name:        "max_redemptions",
		Type:        types.Int(32),
		Nullable:    true,
		Description: "Maximum redemptions",
	},
	{
		Name:        "times_redeemed",
		Type:        types.Int(32),
		Description: "Times redeemed",
	},
	{
		Name:        "created",
		Type:        types.DateTime(),
		Description: "Created",
	},
	{
		Name:        "currency",
		Type:        types.String(),
		Description: "Currency",
	},
	{
		Name:        "duration_in_months",
		Type:        types.Int(32),
		Nullable:    true,
		Description: "Duration in months",
	},
	{
		Name:        "metadata",
		Type:        types.Map(types.String()),
		Description: "Metadata",
	},
	{
		Name:        "valid",
		Type:        types.Boolean(),
		Description: "Still valid",
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
				Description: "Amount tax display",
			},
			{
				Name:        "template",
				Type:        types.String(),
				Nullable:    true,
				Description: "Template",
			},
		}),
		Nullable:    true,
		Description: "Rendering options",
	},
	{
		Name:        "footer",
		Type:        types.String(),
		Nullable:    true,
		Description: "Footer",
	},
	{
		Name: "custom_fields",
		Type: types.Array(types.Object([]types.Property{
			{
				Name:        "name",
				Type:        types.String().WithMaxLength(40),
				Description: "Field name",
			},
			{
				Name:        "value",
				Type:        types.String().WithMaxBytes(140),
				Description: "Field value",
			},
		})),
		Nullable:    true,
		Description: "Custom fields",
	},
})
