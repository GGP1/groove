import React from "react";
import { View } from "react-native";
import { Commentx } from "../../../components";
import { CommonNavProps } from "../../../utils/navigation";

export const App = ({ route }: CommonNavProps<"Comment">) => {
	return (
		<View>
			<Commentx
				comment_id={route.params.comment_id}
				event_id={route.params.event_id}
				user_id={route.params.user_id}
			/>
		</View>
	);
};
