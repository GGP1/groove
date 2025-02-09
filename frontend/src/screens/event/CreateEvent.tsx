import React from "react";
import { View } from "react-native";
// import * as yup from "yup";

// const createEventSchema = yup.object({
// 	name: yup.string().required().min(3).max(60),
// 	description: yup.string().max(150),
// 	public: yup.bool().required(),
// 	virtual: yup.bool().required(),
// 	type: yup.number().required().min(0).max(27),
// 	logo_url: yup.string().url().max(240),
// 	header_url: yup.string().url().max(240),
// 	url: yup.string().url().max(240),
// 	slots: yup.number().required().min(-1),
// 	ticket_type: yup.number().required().min(0).max(4),
// 	min_age: yup.number().required().min(0).max(120),
// 	start_date: yup.date().required().min(new Date()).max(new Date(2200, 1)),
// 	end_date: yup.date().required().when(
// 		"start_date",
// 		(start_date, y) => start_date && y.min(start_date, "End date cannot be before start date"),
// 	),
// 	cron: yup.string().required(), // Should be built by the application based on different user inputs, validate with a regexp
// 	latitude: yup.number().min(-90).max(90),
// 	longitude: yup.number().min(-180).max(180),
// 	address: yup.string().max(120),
// });

export const App = () => {
	return <View />;
};
