import { Formik } from "formik";
import React from "react";
import { StyleSheet, Text, View } from "react-native";
import * as yup from "yup";
import { API } from "../../../api/api";
import { Input, PressableButton } from "../../../components";
import { FormErr } from "../../../components/FormErr";
import { CreatePost } from "../../../types/post";
import { CommonNavProps } from "../../../utils/navigation";

const createPostSchema = yup.object({
	content: yup.string().required().trim().max(40),
	media: yup.array().required(),
});

export const App = ({ route }: CommonNavProps<"CreatePost">) => {
	const createPost = async (post: CreatePost) => {
		try {
			await API.Events.CreatePost(route.params.event_id, post);
		} catch (err) {
			console.log(err);
		}
	};

	return (
		<View>
			<Text style={styles.title}>Register</Text>

			<Formik
				initialValues={{ content: "", media: [] }}
				validationSchema={createPostSchema}
				onSubmit={(values: CreatePost, actions) => {
					actions.resetForm();
					createPost(values);
				}}
			>

				{fk => (
					<View>
						<Input
							placeholder=""
							onChangeText={fk.handleChange("content")}
							onBlur={fk.handleBlur("content")}
							value={fk.values.content}
							autoCapitalize={"none"}
							autoCompleteType={"name"}
							spellCheck={false}
						/>
						<FormErr cond={fk.touched.content} err={fk.errors.content} />

						{/* TODO: select media from device (or urls?) */}

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
