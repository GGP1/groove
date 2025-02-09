import React from "react";
import { View } from "react-native";
import { Postx } from "../../../components";
import { CommonNavProps } from "../../../utils/navigation";

export const App = ({ route }: CommonNavProps<"Post">) => {
	return (
		<View>
			<Postx
				post_id={route.params.post_id}
				event_id={route.params.event_id}
				event_name={route.params.event_name}
				event_logo_url={route.params.event_logo_url}
			/>
		</View>
	);
};
