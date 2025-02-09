import { Formik } from "formik";
import React, { useEffect, useState } from "react";
import { StyleSheet, View } from "react-native";
import MultiSelect from "react-native-multiple-select";
import * as yup from "yup";
import { API } from "../../../api/api";
import { Input, PressableButton } from "../../../components";
import { FormErr } from "../../../components/FormErr";
import { Permission } from "../../../types/permission";
import { Zone } from "../../../types/zone";
import { CommonNavProps } from "../../../utils/navigation";

const createZoneSchema = yup.object({
	name: yup.string().required().trim().max(60),
	required_permission_keys: yup.array(),
});

export const App = ({ route, navigation }: CommonNavProps<"CreateZone">) => {
	const [permissionKeys, setPermissionKeys] = useState<Permission[]>();

	useEffect(() => {
		const getPermissionKeys = async () => {
			try {
				const pkeys = await API.Events.GetPermissions(route.params.event_id);
				setPermissionKeys(pkeys);
			} catch (err) {
				console.log(err);
			}
		};
		getPermissionKeys();
	}, [route.params.event_id]);

	const createZone = async (zone: Zone) => {
		try {
			await API.Events.CreateZone(route.params.event_id, zone);
		} catch (err) {
			console.log(err);
		}
	};

	return (
		<View>
			<Formik
				initialValues={{ name: "", required_permission_keys: [] }}
				validationSchema={createZoneSchema}
				onSubmit={(values: Zone) => {
					createZone(values);
					navigation.goBack();
				}}
			>

				{fk => (
					<View>
						<Input
							placeholder="Name"
							onChangeText={fk.handleChange("name")}
							onBlur={fk.handleBlur("name")}
							value={fk.values.name}
							autoCapitalize="none"
							autoCompleteType="name"
							autoCorrect={false}
						/>
						<FormErr cond={fk.touched.name} err={fk.errors.name} />

						<MultiSelect
							selectText="Pick the permission keys required to enter the zone"
							searchInputPlaceholderText="Search"
							items={permissionKeys ? permissionKeys : []}
							displayKey="name"
							selectedItems={fk.values.required_permission_keys}
							onSelectedItemsChange={(selectedItems) => {
								const keys: string[] = [];
								selectedItems.map((item: Permission) => {
									keys.push(item.key);
								});
								fk.setFieldValue("required_permission_keys", keys);
							}}
						/>

						<PressableButton title="Create zone" onPress={fk.handleSubmit} style={styles.button} />
					</View>
				)}
			</Formik>
		</View>
	);
};

const styles = StyleSheet.create({
	title: {
		fontSize: 17,
		textAlign: "center",
		marginBottom: 15,
	},
	button: {
		alignSelf: "center",
		marginTop: 10,
		marginBottom: 0,
		width: 150,
	},
});
