import React from "react";
import { View } from "react-native";
import { Userx } from "../../components";
import { CommonNavProps } from "../../utils/navigation";

export const App = ({ route }: CommonNavProps<"User">) => {
	return (
		<View>
			<Userx
				id={route.params.user_id}
				user={route.params.item}
			/>
		</View>
	);
};
