import { Formik } from "formik";
import React from "react";
import { StyleSheet, View } from "react-native";
import MultiSelect from "react-native-multiple-select";
import * as yup from "yup";
import { API } from "../../../api/api";
import { Input, PressableButton } from "../../../components";
import { FormErr } from "../../../components/FormErr";
import { Permission } from "../../../types/permission";
import { UpdateRole } from "../../../types/role";
import { CommonNavProps } from "../../../utils/navigation";

const createRoleSchema = yup.object({
	name: yup.string().trim().min(1).max(60),
	permission_keys: yup.array(),
});

export const App = ({ route, navigation }: CommonNavProps<"UpdateRole">) => {
	const role = route.params.role;

	const updateRole = async (r: UpdateRole) => {
		try {
			await API.Events.UpdateRole(route.params.event_id, role.name, r);
		} catch (err) {
			console.log(err);
		}
	};

	return (
		<View>
			<Formik
				initialValues={{ permission_keys: [] }}
				validationSchema={createRoleSchema}
				onSubmit={(values: UpdateRole) => {
					updateRole(values);
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
							selectText="Pick permission keys"
							searchInputPlaceholderText="Search"
							items={role.permission_keys}
							displayKey="name"
							selectedItems={fk.values.permission_keys}
							onSelectedItemsChange={(selectedItems) => {
								const keys: string[] = [];
								selectedItems.map((item: Permission) => {
									keys.push(item.key);
								});
								fk.setFieldValue("permission_keys", keys);
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
