import { Formik } from "formik";
import React from "react";
import { StyleSheet, Switch, Text, View } from "react-native";
import * as yup from "yup";
import { API } from "../../api/api";
import { Input, PressableButton } from "../../components";
import { FormErr } from "../../components/FormErr";
import { UpdateUser } from "../../types/user";
import { MyProfileNavProps } from "../../utils/navigation";

const createUserSchema = yup.object({
	name: yup.string().trim().min(1).max(60),
	username: yup.string().trim().min(1).max(60),
	profile_image_url: yup.string().url().trim().max(240),
	private: yup.bool(),
	invitations: yup.number().min(1).max(2),
});

export const App = ({ route, navigation }: MyProfileNavProps<"UpdateUser">) => {
	const user = route.params.user;

	const updateUser = async (u: UpdateUser) => {
		try {
			await API.Users.Update(user.id, u);
		} catch (err) {
			console.log(err);
		}
	};

	return (
		<View>
			<Formik
				initialValues={{
					name: user.name,
					username: user.username,
					profile_image_url: user.profile_image_url,
					private: user.private,
					invitations: user.invitations,
				}}
				validationSchema={createUserSchema}
				onSubmit={(values: UpdateUser) => {
					updateUser(values);
					navigation.goBack();
				}}
			>

				{fk => (
					<View style={styles.form}>
						<View style={styles.input}>
							<Text>Name</Text>
							<Input
								placeholder="Name"
								value={fk.values.name}
								onChangeText={fk.handleChange("name")}
								autoCompleteType="name"
								spellCheck={false}
							/>
						</View>
						<FormErr cond={fk.touched.name} err={fk.errors.name} />

						<View style={styles.input}>
							<Text>Username</Text>
							<Input
								placeholder="Username"
								value={fk.values.username}
								onChangeText={fk.handleChange("username")}
								spellCheck={false}
							/>
						</View>
						<FormErr cond={fk.touched.username} err={fk.errors.username} />

						<View style={styles.input}>
							<Text>Private</Text>
							<Switch
								onValueChange={(value) => fk.setFieldValue("private", value)}
								value={fk.values.private}
							/>
						</View>

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
	form: {},
	input: {
		flexDirection: "row",
		justifyContent: "space-between",
		alignItems: "center",
		margin: 5,
	},
});
