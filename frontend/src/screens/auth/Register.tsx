import DateTimePicker, { AndroidEvent } from "@react-native-community/datetimepicker";
import { Formik } from "formik";
import React, { useState } from "react";
import { Alert, Keyboard, Platform, StyleSheet, Text, TouchableWithoutFeedback, View } from "react-native";
import { Button } from "react-native-elements";
import { Checkbox } from "react-native-paper";
import * as yup from "yup";
import { API } from "../../api/api";
import { APIKey } from "../../api/key";
import { Center, Input, PasswordInput } from "../../components";
import { FormErr } from "../../components/FormErr";
import { CreateUser, Type } from "../../types/user";
import { AuthNavProps } from "../../utils/navigation";

// https://www.youtube.com/watch?v=urzVC5Zr-JM
// https://www.youtube.com/watch?v=ftLy78R8xrg
// https://www.youtube.com/watch?v=o_ErcEKV23I
// https://docs.nativebase.io/login-signup-forms

const createUserSchema = yup.object({
	name: yup.string().required().trim(),
	email: yup.string().required().email("invalid email").trim(),
	username: yup.string().required().max(24).lowercase().trim().
		// [a-zA-Z0-9._] contains lowercases, uppercases, numbers or (._) only
		matches(/^[a-zA-Z0-9._]+$/, "invalid username"),
	password: yup.string().required().min(10).trim().
		// (?=.*[a-z]): at least 1 lowercase
		// (?=.*[A-Z]): at least 1 uppercase
		// (?=.*\d): at least 1 digit
		// {10,}: 10 or more characters
		matches(/^(?=.*[a-z])(?=.*[A-Z])(?=.*\d).{10,}$/,
			"invalid password it must: • contain 10 or more characters\n• one lowercase\n•one uppercase\n•one number"),
	birth_date: yup.date().required().min(new Date(1910, 1)).max(new Date()),
	type: yup.number().required().test("type-validation", "invalid type", function (value) {
		return value === Type.Business || value === Type.Personal;
	}),
});

export const App = ({ navigation }: AuthNavProps<"Register">) => {
	const [showDatePicker, setShowDatePicker] = useState<boolean>(false);
	const [checked, setChecked] = useState<boolean>(false);
	const [confirmedPassword, setConfirmedPassword] = useState<string>();
	const [confirmedBlur, setConfirmedBlur] = useState<boolean>(false);

	const register = async (createUser: CreateUser) => {
		try {
			const res = await API.Users.Create(createUser);
			if (res.id) {
				await APIKey.store(res);
				navigation.navigate("Login");
			}
		} catch (err) {
			console.log("register:", err);
		}
	};

	return (
		<Center>
			<TouchableWithoutFeedback onPress={Keyboard.dismiss}>
				<View>
					<Text style={styles.title}>Register</Text>
					<Formik
						initialValues={{ name: "", email: "", username: "", password: "", birth_date: new Date(), type: Type.Personal }}
						validationSchema={createUserSchema}
						onSubmit={(values: CreateUser, actions) => {
							if (!confirmedPassword || confirmedPassword !== values.password) {
								Alert.alert("Error", "Passwords do not match");
								return;
							}
							setConfirmedPassword(undefined);
							setConfirmedBlur(false);
							actions.resetForm();
							register(values);
						}}
					>
						{fk => (
							<View>
								<Input
									placeholder="Name"
									onChangeText={fk.handleChange("name")}
									onBlur={fk.handleBlur("name")}
									value={fk.values.name}
									autoCapitalize={"words"}
									autoCompleteType={"name"}
									spellCheck={false}
								/>
								<FormErr cond={fk.touched.name} err={fk.errors.name} />

								<Input
									placeholder="Email"
									onChangeText={fk.handleChange("email")}
									onBlur={fk.handleBlur("email")}
									value={fk.values.email}
									autoCapitalize={"none"}
									autoCompleteType={"email"}
									spellCheck={false}
								/>
								<FormErr cond={fk.touched.email} err={fk.errors.email} />

								<Input
									placeholder="Username"
									onChangeText={fk.handleChange("username")}
									onBlur={fk.handleBlur("username")}
									value={fk.values.username}
									autoCapitalize={"none"}
									autoCompleteType={"username"}
									spellCheck={false}
								/>
								<FormErr cond={fk.touched.username} err={fk.errors.username} />

								<PasswordInput
									onChangeText={fk.handleChange("password")}
									onBlur={fk.handleBlur("password")}
									value={fk.values.password}
								/>
								<FormErr cond={fk.touched.password} err={fk.errors.password} />

								<PasswordInput
									placeholder="Confirm password"
									onChangeText={(text) => setConfirmedPassword(text)}
									onBlur={() => setConfirmedBlur(true)}
									value={confirmedPassword}
								/>
								<FormErr
									cond={confirmedBlur && (confirmedPassword !== fk.values.password)}
									err={"passwords do not match"}
								/>

								<View style={styles.checkbox}>
									<Checkbox status={checked ? "checked" : "unchecked"} onPress={() => {
										setChecked(!checked);
										fk.setFieldValue("type", checked ? Type.Business : Type.Personal);
									}} />
									<Text style={styles.checkboxText}>Business</Text>
								</View>

								<Button onPress={() => setShowDatePicker(true)} title="Select birth date" style={styles.button} />
								{showDatePicker && (
									<DateTimePicker
										value={fk.values.birth_date}
										display={"spinner"} // TODO: try "inline" in ios
										minimumDate={new Date(1910, 1)}
										maximumDate={new Date()}
										onChange={(_: AndroidEvent, selectedDate: Date | undefined) => {
											setShowDatePicker(Platform.OS === "ios");
											fk.setFieldValue("birth_date", selectedDate);
										}}
									/>
								)}
								<Button title="Register" onPress={fk.handleSubmit} style={styles.button} />
							</View>
						)}
					</Formik>
				</View>
			</TouchableWithoutFeedback>
		</Center>
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
	checkbox: {
		alignSelf: "center",
		flexDirection: "row",
	},
	checkboxText: {
		alignSelf: "center",
	},
});

