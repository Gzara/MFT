package tfhelper

import (
	"context"
	"reflect"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func ModelToSchema(ctx context.Context, modelName string, model interface{}) schema.Schema {
	return schema.Schema{Attributes: _modelToAttributes(ctx, modelName, reflect.TypeOf(reflect.ValueOf(model).Elem().Interface()))}
}

func _modelToAttributes(ctx context.Context, modelName string, value reflect.Type) map[string]schema.Attribute {
	attrs := make(map[string]schema.Attribute)
	switch value.Kind() {
	case reflect.Struct:
		// v := value.Interface()
		// first := true
		// reflectType := reflect.TypeOf(v)
		reflectType := value
		// reflectValue := reflect.ValueOf(v)
		for i := 0; i < reflectType.NumField(); i++ {
			fieldName := reflectType.Field(i).Name
			tag := reflectType.Field(i).Tag
			tfsdk := tag.Get(tfsdkTagName)
			flags := tag.Get(helperTagName)
			name := FlagsTfsdkGetName(tfsdk)

			mustCheckSupportedAttributes(modelName+"."+fieldName, flags)
			required := FlagsHas(flags, "required")
			computed := FlagsHas(flags, "computed")
			sensitive := FlagsHas(flags, "sensitive")
			elementtype, _ := FlagsGet(flags, "elementtype")

			state := FlagsHas(flags, "state")
			def, defok := FlagsGet(flags, "default")
			var s []planmodifier.String = nil
			if state {
				s = []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				}
			}
			optional := (!required && !computed) || FlagsHas(flags, "optional") || defok
			computed = computed || defok

			// typ := reflectType.Field(i).Type.
			typestr := reflectType.Field(i).Type.String()
			kind := reflectType.Field(i).Type.Kind()
			tflog.Info(ctx, ">>"+fieldName+" : "+typestr+"/"+kind.String())

			switch kind {
			case reflect.Slice:
				t := reflectType.Field(i).Type.Elem()
				typestr := t.String()
				kind2 := t.Kind()

				switch kind2 {
				case reflect.Struct:
					if strings.HasPrefix(typestr, "basetypes.") {
						switch typestr {
						case "basetypes.StringValue":
							attrs[name] = schema.ListAttribute{
								ElementType: types.StringType,
								Required:    required,
								Optional:    optional,
								Computed:    computed,
								Sensitive:   sensitive,
							}
						default:
							panic("unsupported slice type: " + typestr + "(" + modelName + "." + fieldName + ")")
						}
					} else {
						tflog.Info(ctx, ">>"+fieldName+" : "+typestr+"/"+kind.String())
						attrs[name] = schema.ListNestedAttribute{
							NestedObject: schema.NestedAttributeObject{
								Attributes: _modelToAttributes(ctx, modelName+"."+fieldName, reflectType.Field(i).Type.Elem()),
							},
							Required:  required,
							Optional:  optional,
							Computed:  computed,
							Sensitive: sensitive,
						}
					}
				default:
					panic("unsupported: Slice" + kind2.String() + " (" + modelName + "." + fieldName + ")")
				}

			case reflect.Struct:
				if strings.HasPrefix(typestr, "basetypes.") {
					switch typestr {
					case "basetypes.MapValue":
						if elementtype == "string" {
							attrs[name] = schema.MapAttribute{
								ElementType: types.StringType,
								Required:    required,
								Optional:    optional,
								Computed:    computed,
								Sensitive:   sensitive,
							}
						} else {
							panic("unsupported element type: '" + elementtype + "' (" + modelName + "." + fieldName + ")")
						}
					case "basetypes.ObjectValue":
						elementModel := registeredTypes[elementtype]
						if elementModel == nil {
							panic("unsupported element type: '" + elementtype + "' (" + modelName + "." + fieldName + ")")
						}
						elementAttrs := _modelToAttributes(ctx, modelName+"."+fieldName, reflect.TypeOf(reflect.ValueOf(elementModel).Elem().Interface()))
						attrs[name] = schema.SingleNestedAttribute{
							Attributes: elementAttrs,
							Required:   required,
							Optional:   optional,
							Computed:   computed,
							Sensitive:  sensitive,
						}
					case "basetypes.ListValue":
						if elementtype == "string" {
							var d defaults.List
							if defok && def == "" {
								f := types.ListNull(types.StringType)
								d = listdefault.StaticValue(f)
							} else if defok {
								panic("unsupported default value: " + def + "(" + modelName + "." + fieldName + ")")
							}
							attrs[name] = schema.ListAttribute{
								ElementType: types.StringType,
								Required:    required,
								Optional:    optional,
								Computed:    computed,
								Sensitive:   sensitive,
								Default:     d,
							}
						} else {
							elementModel := registeredTypes[elementtype]
							if elementModel == nil {
								panic("unsupported element type: '" + elementtype + "' (" + modelName + "." + fieldName + ")")
							}
							elementAttrs := _modelToAttributes(ctx, modelName+"."+fieldName, reflect.TypeOf(reflect.ValueOf(elementModel).Elem().Interface()))

							var d defaults.List
							if defok && def == "" {
								f := types.ListNull(types.ObjectType{})
								d = listdefault.StaticValue(f)
							} else if defok {
								panic("unsupported default value: " + def + "(" + modelName + "." + fieldName + ")")
							}
							attrs[name] = schema.ListNestedAttribute{
								NestedObject: schema.NestedAttributeObject{
									Attributes: elementAttrs,
								},
								Required:  required,
								Optional:  optional,
								Computed:  computed,
								Sensitive: sensitive,
								Default:   d,
							}
						}
					/*case "basetypes.ObjectValue":
					attrs[name] = schema.MapAttribute{
						ElementType: types.StringType,
						Required:    required,
						Optional:    optional,
						Computed:    computed,
						Sensitive:   sensitive,
					}*/
					case "basetypes.StringValue":
						var d defaults.String
						if defok {
							d = stringdefault.StaticString(def)
						}

						attrs[name] = schema.StringAttribute{
							Required:      required,
							Optional:      optional,
							Computed:      computed,
							Sensitive:     sensitive,
							Default:       d,
							PlanModifiers: s,
						}
					case "basetypes.BoolValue":
						var d defaults.Bool
						if defok && def != "" {
							if def == "true" {
								d = booldefault.StaticBool(true)
							} else if def == "false" {
								d = booldefault.StaticBool(false)
							} else {
								panic("unsupported default value: " + def + "(" + modelName + "." + fieldName + ")")
							}
						} else if defok && def == "" {
							d = booldefault.StaticBool(false)
						}
						attrs[name] = schema.BoolAttribute{
							Required:  required,
							Optional:  optional,
							Computed:  computed,
							Sensitive: sensitive,
							Default:   d,
						}
					case "basetypes.Int64Value":
						var d defaults.Int64
						if defok && def != "" {
							i, err := strconv.ParseInt(def, 10, 64)
							if err != nil {
								panic(err)
							}
							d = int64default.StaticInt64(i)
						} else if defok && def == "" {
							d = int64default.StaticInt64(0)
						}
						attrs[name] = schema.Int64Attribute{
							Required:  required,
							Optional:  optional,
							Computed:  computed,
							Sensitive: sensitive,
							Default:   d,
						}
					default:
						panic("unsupported type: " + typestr + "(" + modelName + "." + fieldName + ")")
					}
				} else {
					tflog.Info(ctx, ">>"+fieldName+" : "+typestr+"/"+kind.String())
					attrs[name] = schema.SingleNestedAttribute{
						Attributes: _modelToAttributes(ctx, modelName+"."+fieldName, reflectType.Field(i).Type),
						Required:   required,
						Optional:   optional,
						Computed:   computed,
						Sensitive:  sensitive,
					}
				}
			case reflect.Ptr:
				tflog.Info(ctx, ">>"+fieldName+" : "+typestr+"/"+kind.String())
				attrs[name] = schema.SingleNestedAttribute{
					Attributes: _modelToAttributes(ctx, modelName+"."+fieldName, reflectType.Field(i).Type.Elem()),
					Required:   required,
					Optional:   optional,
					Computed:   computed,
					Sensitive:  sensitive,
				}
			default:
				panic("unsupported: type" + kind.String() + " (" + modelName + "." + fieldName + ")")
			}
		}
	default:
		panic("unsupported: type" + value.Kind().String() + " (" + modelName + ")")
	}
	return attrs
}