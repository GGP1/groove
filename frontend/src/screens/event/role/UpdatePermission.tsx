import { Formik } from "formik";
import React from "react";
import { StyleSheet, View } from "react-native";
import * as yup from "yup";
import { API } from "../../../api/api";
import { Input, PressableButton } from "../../../components";
import { FormErr } from "../../../components/FormErr";
import { UpdatePermission } from "../../../types/permission";
import { CommonNavProps } from "../../../utils/navigation";

const createPermissionSchema = yup.object({
	name: yup.string().required().trim().min(1).max(60),
	description: yup.string().max(200),
	key: yup.string().required().trim().max(30).matches(/^[a-z]+(?:_[a-z]+)*$/),
});

export const App = ({ route, navigation }: CommonNavProps<"UpdatePermission">) => {
	const permission = route.params.permission;

	const updatePermission = async (p: UpdatePermission) => {
		try {
			await API.Events.UpdatePermission(route.params.event_id, permission.key, p);
		} catch (err) {
			console.log(err);
		}
	};

	return (
		<View>
			<Formik
				initialValues={{ name: permission.name, description: permission.description }}
				validationSchema={createPermissionSchema}
				onSubmit={(values: UpdatePermission) => {
					updatePermission(values);
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

						<Input
							placeholder="Description"
							onChangeText={fk.handleChange("description")}
							onBlur={fk.handleBlur("description")}
							value={fk.values.description}
							autoCapitalize="none"
							autoCorrect={true}
						/>
						<FormErr cond={fk.touched.description} err={fk.errors.description} />

						<Input
							placeholder="Key"
							onChangeText={fk.handleChange("key")}
							onBlur={fk.handleBlur("key")}
							value={fk.values.key}
							autoCapitalize="none"
							autoCompleteType="off"
							autoCorrect={false}
						/>
						<FormErr cond={fk.touched.key} err={fk.errors.key} />

						<PressableButton title="Create" onPress={fk.handleSubmit} style={styles.button} />
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
