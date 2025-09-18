//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package stripe

import "github.com/meergo/meergo/core/types"

// https://docs.stripe.com/api/customers/create and
// https://docs.stripe.com/api/customers/update.
//
// The "test_clock" field has been excluded because it is not relevant to Meergo.

var destinationSchema = types.Object([]types.Property{
	{
		Name:     "address",
		Type:     destinationAddress,
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
		// Only when creating.
		Name:        "payment_method",
		Type:        types.Text(),
		Nullable:    true,
		Description: "Used only when creating a customer, ignored when updating.",
	},
	{
		Name:     "phone",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name:     "shipping",
		Type:     destinationShipping,
		Nullable: true,
	},
	{
		Name: "tax",
		Type: destinationTax,
	},
	{
		Name: "balance",
		Type: types.Int(64),
	},
	{
		Name: "cash_balance",
		Type: types.Object([]types.Property{
			{
				Name: "settings",
				Type: types.Object([]types.Property{
					{
						Name:     "reconciliation_mode",
						Type:     types.Text().WithValues("automatic", "manual", "merchant_default"),
						Nullable: true,
					},
				}),
				Nullable: true,
			},
		}),
	},
	{
		// Only when updating.
		Name:        "default_source",
		Type:        types.Text(),
		Description: "Used only when updating a customer, ignored when creating.",
	},
	{
		Name: "invoice_prefix",
		Type: types.Text(),
	},
	{
		Name: "invoice_settings",
		Type: destinationInvoiceSettings,
	},
	{
		// If the Stripe account applies account-level sequencing, this
		// parameter is ignored in API requests and excluded from API responses.
		Name: "next_invoice_sequence",
		Type: types.Int(64),
	},
	{
		Name: "preferred_locales",
		Type: types.Array(types.Text()),
	},
	{
		Name: "source",
		Type: types.Text(),
	},
	{
		Name:     "tax_exempt",
		Type:     types.Text().WithValues("none", "exempt", "reverse"),
		Nullable: true,
	},
	{
		// Only when creating.
		Name: "tax_id_data",
		Type: types.Array(types.Object([]types.Property{
			{
				Name:           "type",
				Type:           types.Text(),
				CreateRequired: true,
			},
			{
				Name:           "value",
				Type:           types.Text(),
				CreateRequired: true,
			},
		})),
		Description: "Used only when creating a customer, ignored when updating.",
	},
})

var destinationAddress = types.Object([]types.Property{
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

var destinationShipping = types.Object([]types.Property{
	{
		Name:           "address",
		Type:           destinationAddress,
		CreateRequired: true,
		UpdateRequired: true,
	},
	{
		Name:           "name",
		Type:           types.Text(),
		CreateRequired: true,
		UpdateRequired: true,
	},
	{
		Name:     "phone",
		Type:     types.Text(),
		Nullable: true,
	},
})

var destinationTax = types.Object([]types.Property{
	{
		Name: "ip_address",
		Type: types.Text(),
	},
	{
		Name: "validate_location",
		Type: types.Text(), // don't add enum values, as they differ from creation to update.
	},
})

var destinationInvoiceSettings = types.Object([]types.Property{
	{
		Name: "custom_fields",
		Type: types.Array(types.Object([]types.Property{
			{
				Name:           "name",
				Type:           types.Text(),
				Nullable:       true,
				CreateRequired: true,
				UpdateRequired: true,
			},
			{
				Name:           "value",
				Type:           types.Text(),
				Nullable:       true,
				CreateRequired: true,
				UpdateRequired: true,
			},
		})).WithMaxElements(4),
		Nullable: true,
	},
	{
		Name:     "default_payment_method",
		Type:     types.Text(),
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
				Type:     types.Text().WithValues("exclude_tax", "include_inclusive_tax"),
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
