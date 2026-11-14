// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package stripe

import (
	"github.com/krenalis/krenalis/tools/types"
)

// https://docs.stripe.com/api/customers/create and
// https://docs.stripe.com/api/customers/update.
//
// The "source" and "test_clock" fields has been excluded because it is not relevant to Krenalis.

var destinationSchema = types.Object([]types.Property{
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
		Type:        destinationAddress,
		Nullable:    true,
		DisplayName: "Billing details",
	},
	{
		Name:        "shipping",
		Type:        destinationShipping,
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
		DisplayName: "Locales",
	},
	{
		Name:        "invoice_prefix",
		Type:        types.String().WithMaxBytes(12),
		DisplayName: "Invoice prefix",
	},
	{
		// If the Stripe account applies account-level sequencing, this
		// parameter is ignored in API requests and excluded from API responses.
		Name:        "next_invoice_sequence",
		Type:        types.Int(32).WithIntRange(1, 1000000000),
		DisplayName: "Next invoice sequence",
	},
	{
		Name:        "tax_exempt",
		Type:        types.String().WithValues("none", "exempt", "reverse"),
		Nullable:    true,
		DisplayName: "Tax status",
	},
	{
		// Only when creating.
		Name: "tax_id_data",
		Type: types.Array(types.Object([]types.Property{
			{
				Name:           "type",
				Type:           types.String().WithValues("ad_nrt", "ae_trn", "al_tin", "am_tin", "ao_tin", "ar_cuit", "au_abn", "au_arn", "aw_tin", "az_tin", "ba_tin", "bb_tin", "bd_bin", "bf_ifu", "bg_uic", "bh_vat", "bj_ifu", "bo_tin", "br_cnpj", "br_cpf", "bs_tin", "by_tin", "ca_bn", "ca_gst_hst", "ca_pst_bc", "ca_pst_mb", "ca_pst_sk", "ca_qst", "cd_nif", "ch_uid", "ch_vat", "cl_tin", "cm_niu", "cn_tin", "co_nit", "cr_tin", "cv_nif", "de_stn", "do_rcn", "ec_ruc", "eg_tin", "es_cif", "et_tin", "eu_oss_vat", "eu_vat", "gb_vat", "ge_vat", "gn_nif", "hk_br", "hr_oib", "hu_tin", "id_npwp", "il_vat", "in_gst", "is_vat", "jp_cn", "jp_rn", "jp_trn", "ke_pin", "kg_tin", "kh_tin", "kr_brn", "kz_bin", "la_tin", "li_uid", "li_vat", "ma_vat", "md_vat", "me_pib", "mk_vat", "mr_nif", "mx_rfc", "my_frp", "my_itn", "my_sst", "ng_tin", "no_vat", "no_voec", "np_pan", "nz_gst", "om_vat", "pe_ruc", "ph_tin", "ro_tin", "rs_pib", "ru_inn", "ru_kpp", "sa_vat", "sg_gst", "sg_uen", "si_tin", "sn_ninea", "sr_fin", "sv_nit", "th_vat", "tj_tin", "tr_tin", "tw_vat", "tz_vat", "ua_vat", "ug_tin", "us_ein", "uy_ruc", "uz_tin", "uz_vat", "ve_rif", "vn_tin", "za_vat", "zm_tin", "zw_tin"),
				CreateRequired: true,
				DisplayName:    "ID type",
			},
			{
				Name:           "value",
				Type:           types.String(),
				CreateRequired: true,
				DisplayName:    "ID value",
			},
		})),
		DisplayName: "Tax ID",
		Description: "Ignored when updating a customer",
	},
	{
		Name:        "metadata",
		Type:        types.Map(types.String()),
		DisplayName: "Metadata",
	},
	{
		// Only when creating.
		Name:        "payment_method",
		Type:        types.String(),
		Nullable:    true,
		DisplayName: "Payment method",
		Description: "Ignored when updating a customer",
	},
	{
		// Only when updating.
		Name:        "default_source",
		Type:        types.String(),
		DisplayName: "Default payment method",
		Description: "Ignored when creating a customer",
	},
	{
		Name:        "balance",
		Type:        types.Int(64).WithIntRange(-1000000000000, 1000000000000),
		DisplayName: "Balance",
	},
	{
		Name: "cash_balance",
		Type: types.Object([]types.Property{
			{
				Name: "settings",
				Type: types.Object([]types.Property{
					{
						Name:        "reconciliation_mode",
						Type:        types.String().WithValues("automatic", "manual", "merchant_default"),
						Nullable:    true,
						DisplayName: "Reconciliation mode",
					},
				}),
				Nullable:    true,
				DisplayName: "Settings",
			},
		}),
		DisplayName: "Balance settings",
	},
	{
		Name:        "tax",
		Type:        destinationTax,
		DisplayName: "Tax details",
	},
	{
		Name:        "invoice_settings",
		Type:        destinationInvoiceSettings,
		DisplayName: "Invoice settings",
	},
})

var destinationAddress = types.Object([]types.Property{
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

var destinationShipping = types.Object([]types.Property{
	{
		Name:           "name",
		Type:           types.String(),
		CreateRequired: true,
		UpdateRequired: true,
		DisplayName:    "Customer name",
	},
	{
		Name:           "address",
		Type:           destinationAddress,
		CreateRequired: true,
		UpdateRequired: true,
		DisplayName:    "Address",
	},
	{
		Name:        "phone",
		Type:        types.String(),
		Nullable:    true,
		DisplayName: "Phone number",
	},
})

var destinationTax = types.Object([]types.Property{
	{
		Name:        "ip_address",
		Type:        types.String(),
		DisplayName: "Recent customer IP",
	},
	{
		Name:        "validate_location",
		Type:        types.String().WithValues("auto", "deferred", "immediately"), // "auto" is only allowed during update
		DisplayName: "Location validation timing",
		Description: "\"auto\" is ignored when creating a customer",
	},
})

var destinationInvoiceSettings = types.Object([]types.Property{
	{
		Name:        "default_payment_method",
		Type:        types.String(),
		Nullable:    true,
		DisplayName: "Default payment method",
	},
	{
		Name: "rendering_options",
		Type: types.Object([]types.Property{
			{
				Name:        "amount_tax_display",
				Type:        types.String().WithValues("exclude_tax", "include_inclusive_tax"),
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
				Name:           "name",
				Type:           types.String().WithMaxLength(40),
				Nullable:       true,
				CreateRequired: true,
				UpdateRequired: true,
				DisplayName:    "Field name",
			},
			{
				Name:           "value",
				Type:           types.String().WithMaxBytes(140),
				Nullable:       true,
				CreateRequired: true,
				UpdateRequired: true,
				DisplayName:    "Field value",
			},
		})).WithMaxElements(4),
		Nullable:    true,
		DisplayName: "Custom fields",
	},
})
