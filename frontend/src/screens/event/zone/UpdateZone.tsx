import { Formik } from "formik";
import React from "react";
import { StyleSheet, View } from "react-native";
import MultiSelect from "react-native-multiple-select";
import * as yup from "yup";
import { API } from "../../../api/api";
import { Input, PressableButton } from "../../../components";
import { FormErr } from "../../../components/FormErr";
import { Permission } from "../../../types/permission";
import { UpdateZone } from "../../../types/zone";
import { CommonNavProps } from "../../../utils/navigation";

const editZoneSchema = yup.object({
	name: yup.string().trim().min(1).max(60),
	required_permission_keys: yup.array(),
});

export const App = ({ route, navigation }: CommonNavProps<"UpdateZone">) => {
	const zone = route.params.zone;

	const updateZone = async (z: UpdateZone) => {
		try {
			await API.Events.UpdateZone(route.params.event_id, route.params.zone.name, z);
		} catch (err) {
			console.log(err);
		}
	};

	return (
		<View>
			<Formik
				initialValues={{ required_permission_keys: [] }}
				validationSchema={editZoneSchema}
				onSubmit={(values: UpdateZone) => {
					updateZone(values);
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
							spellCheck={false}
						/>
						<FormErr cond={fk.touched.name} err={fk.errors.name} />

						<MultiSelect
							selectText="Pick the permission keys required to enter the zone"
							searchInputPlaceholderText="Search"
							items={zone ? zone.required_permission_keys : []}
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

						<PressableButton title="Save" onPress={fk.handleSubmit} style={styles.button} />
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
