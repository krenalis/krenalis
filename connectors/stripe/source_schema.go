//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package stripe

import "github.com/meergo/meergo/core/types"

// https://docs.stripe.com/api/customers/object.
//
// Currently, we don't support expanded responses/fields. We have an issue about
// them: https://github.com/meergo/meergo/issues/1818.

var sourceSchema = types.Object([]types.Property{
	{
		Name: "id",
		Type: types.Text(),
	},
	{
		Name:     "address",
		Type:     sourceAddress,
		Nullable: true,
	},
	{
		Name:     "description",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name:     "email",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name: "metadata",
		Type: types.Map(types.Text()),
	},
	{
		Name:     "name",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name:     "phone",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name:     "shipping",
		Type:     sourceShipping,
		Nullable: true,
	},
	{
		Name: "object",
		Type: types.Text(),
	},
	{
		Name: "balance",
		Type: types.Int(64),
	},
	{
		Name: "created",
		Type: types.DateTime(),
	},
	{
		Name:     "currency",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name:     "delinquent",
		Type:     types.Boolean(),
		Nullable: true,
	},
	{
		Name:     "discount",
		Type:     sourceDiscount,
		Nullable: true,
	},
	{
		Name:     "invoice_prefix",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name: "invoice_settings",
		Type: sourceInvoiceSettings,
	},
	{
		Name: "livemode",
		Type: types.Boolean(),
	},
	{
		Name:         "next_invoice_sequence",
		Type:         types.Int(64),
		Nullable:     true,
		ReadOptional: true, // if the Stripe account applies account-level sequencing, this parameter is ignored in API requests and excluded from API responses.
	},
	{
		Name:     "preferred_locales",
		Type:     types.Array(types.Text()),
		Nullable: true,
	},
	{
		Name:     "tax_exempt",
		Type:     types.Text().WithValues("none", "exempt", "reverse"),
		Nullable: true,
	},
})

var sourceAddress = types.Object([]types.Property{
	{
		Name:     "city",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name:     "country",
		Type:     types.Text(), // don't limit to 2 chars: ISO 3166-1 alpha-2 is recommended but not enforced by Stripe.
		Nullable: true,
	},
	{
		Name:     "line1",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name:     "line2",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name:     "postal_code",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name:     "state",
		Type:     types.Text(),
		Nullable: true,
	},
})

var sourceShipping = types.Object([]types.Property{
	{
		Name: "address",
		Type: sourceAddress,
	},
	{
		Name: "name",
		Type: types.Text(),
	},
	{
		Name:     "phone",
		Type:     types.Text(),
		Nullable: true,
	},
})

var sourceDiscount = types.Object([]types.Property{
	{
		Name: "id",
		Type: types.Text(),
	},
	{
		Name: "object",
		Type: types.Text(),
	},
	{
		Name:     "checkout_session",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name: "coupon",
		Type: sourceCoupon,
	},
	{
		Name:     "end",
		Type:     types.DateTime(),
		Nullable: true,
	},
	{
		Name:     "invoice",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name:     "invoice_item",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name: "start",
		Type: types.DateTime(),
	},
	{
		Name:     "subscription",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name:     "subscription_item",
		Type:     types.Text(),
		Nullable: true,
	},
})

var sourceCoupon = types.Object([]types.Property{
	{
		Name: "id",
		Type: types.Text(),
	},
	{
		Name: "object",
		Type: types.Text(),
	},
	{
		Name:     "amount_off",
		Type:     types.Int(64),
		Nullable: true,
	},
	{
		Name: "created",
		Type: types.DateTime(),
	},
	{
		Name: "currency",
		Type: types.Text(),
	},
	{
		Name: "duration",
		Type: types.Text().WithValues("forever", "once", "repeating"),
	},
	{
		Name:     "duration_in_months",
		Type:     types.Int(64),
		Nullable: true,
	},
	{
		Name: "livemode",
		Type: types.Boolean(),
	},
	{
		Name:     "max_redemptions",
		Type:     types.Int(64),
		Nullable: true,
	},
	{
		Name: "metadata",
		Type: types.Map(types.Text()),
	},
	{
		Name: "name",
		Type: types.Text(),
	},
	{
		Name: "percent_off",
		Type: types.Float(64),
	},
	{
		Name:     "redeem_by",
		Type:     types.DateTime(),
		Nullable: true,
	},
	{
		Name: "times_redeemed",
		Type: types.Int(64),
	},
	{
		Name: "valid",
		Type: types.Boolean(),
	},
})

var sourceInvoiceSettings = types.Object([]types.Property{
	{
		Name: "custom_fields",
		Type: types.Array(types.Object([]types.Property{
			{
				Name: "name",
				Type: types.Text(),
			},
			{
				Name: "value",
				Type: types.Text(),
			},
		})),
		Nullable: true,
	},
	{
		Name:     "footer",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name: "rendering_options",
		Type: types.Object([]types.Property{
			{
				Name:     "amount_tax_display",
				Type:     types.Text(),
				Nullable: true,
			},
			{
				Name:     "template",
				Type:     types.Text(),
				Nullable: true,
			},
		}),
		Nullable: true,
	},
})
