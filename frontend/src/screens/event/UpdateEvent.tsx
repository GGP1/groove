import React from "react";
import { View } from "react-native";
// import * as yup from "yup";

// const editEventSchema = yup.object({
// 	name: yup.string().min(3).max(60),
// 	description: yup.string().max(150),
// 	type: yup.number().min(0).max(27),
// 	logo_url: yup.string().url().max(240),
// 	header_url: yup.string().url().max(240),
// 	url: yup.string().url().max(240),
// 	slots: yup.number().min(-1),
// 	min_age: yup.number().min(0).max(120),
// 	start_date: yup.date().min(new Date()).max(new Date(2200, 1)),
// 	end_date: yup.date().when(
// 		"start_date",
// 		(start_date, y) => start_date && y.min(start_date, "End date cannot be before start date"),
// 	),
// 	cron: yup.string(), // Should be built by the application based on different user inputs, validate with a regexp
// 	latitude: yup.number().min(-90).max(90),
// 	longitude: yup.number().min(-180).max(180),
// 	address: yup.string().max(120),
// });

export const App = () => {
	return <View />;
};
