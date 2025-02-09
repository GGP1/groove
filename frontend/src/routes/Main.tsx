import { NavigationContainer } from "@react-navigation/native";
import React, { useContext, useEffect, useState } from "react";
import { ActivityIndicator } from "react-native";
import { Center } from "../components";
import { SessionContext } from "../context/Session";
import { Session } from "../utils/session";
import { AuthStack } from "./AuthStack";
import { Tabs } from "./Tabs";

const Main = () => {
	const { user, login } = useContext(SessionContext);
	const [loading, setLoading] = useState(true); // TODO: use context to handle loading everytime something is loading

	useEffect(() => {
		const restoreSession = async () => {
			if (!user) {
				await Session.restore(login);
			}
			setLoading(false);
		};

		restoreSession();
	}, [user, login]);

	// const errorHandler = (error: Error, info: { componentStack: string }) => {
	// 	// TODO: Send error to server
	// 	console.log(error, info.componentStack);
	// };

	if (loading) {
		return (
			<Center>
				<ActivityIndicator size="large" color="#d9d5cd" />
			</Center>
		);
	}
	// <ErrorBoundary
	// 	FallbackComponent={ErrorFallback}
	// 	onReset={() => { /** Reset app state */ }}
	// 	onError={errorHandler}
	// >
	// </ErrorBoundary>

	return (
		<NavigationContainer>
			{user ? <Tabs /> : <AuthStack />}
		</NavigationContainer>
	);
};

export default Main;
