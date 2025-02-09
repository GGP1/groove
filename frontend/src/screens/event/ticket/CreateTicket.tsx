import { Formik } from "formik";
import React from "react";
import { StyleSheet, Text, View } from "react-native";
import { Input } from "react-native-elements";
import * as yup from "yup";
import { API } from "../../../api/api";
import { PressableButton } from "../../../components";
import { FormErr } from "../../../components/FormErr";
import { Ticket } from "../../../types/ticket";
import { CommonNavProps } from "../../../utils/navigation";

const createTicketSchema = yup.object({
	name: yup.string().required().trim().max(60),
	description: yup.string().trim().max(200),
	available_count: yup.number().required().min(0),
	cost: yup.number().required().min(0),
	linked_role: yup.string().trim().max(60),
});

export const App = ({ route, navigation }: CommonNavProps<"CreateTicket">) => {
	const createTicket = async (t: Ticket) => {
		try {
			await API.Events.CreateTicket(route.params.event_id, t);
		} catch (err) {
			console.log(err);
		}
	};

	return (
		<View>
			<Formik
				initialValues={{
					name: "",
					description: "",
					available_count: 0,
					cost: 0,
					linked_role: "",
				}}
				validationSchema={createTicketSchema}
				onSubmit={(values: Ticket) => {
					createTicket(values);
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
								autoCorrect={false}
							/>
						</View>
						<FormErr cond={fk.touched.name} err={fk.errors.name} />

						<View style={styles.input}>
							<Text>Description</Text>
							<Input
								placeholder="Description"
								value={fk.values.description}
								onChangeText={fk.handleChange("description")}
								autoCorrect={true}
							/>
						</View>
						<FormErr cond={fk.touched.description} err={fk.errors.description} />

						<View style={styles.input}>
							<Text>Available count</Text>
							<Input
								placeholder="Available count"
								value={fk.values.available_count.toString()}
								onChangeText={fk.handleChange("available_count")}
								autoCorrect={false}
								keyboardType="decimal-pad"
							/>
						</View>
						<FormErr cond={fk.touched.available_count} err={fk.errors.available_count} />

						<View style={styles.input}>
							<Text>Cost</Text>
							<Input
								placeholder="Cost"
								value={fk.values.cost.toString()}
								onChangeText={fk.handleChange("cost")}
								autoCorrect={false}
								keyboardType="decimal-pad"
							/>
						</View>
						<FormErr cond={fk.touched.cost} err={fk.errors.cost} />

						<View style={styles.input}>
							<Text>Linked role</Text>
							<Input
								placeholder="Linked role"
								value={fk.values.linked_role}
								onChangeText={fk.handleChange("linked_role")}
								autoCompleteType="name"
								autoCorrect={false}
							/>
						</View>
						<FormErr cond={fk.touched.linked_role} err={fk.errors.linked_role} />

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
