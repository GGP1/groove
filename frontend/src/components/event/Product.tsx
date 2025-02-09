import React from "react";
import { StyleSheet, Text, View } from "react-native";
import { Product } from "../../types/product";

interface Props {
	product: Product
}

/** Productx .. */
export const Productx = (props: Props) => {
	const { product } = props;

	return (
		<View style={styles.container}>
			<Text>
				{product.id},
				{product.stock},
				{product.brand},
				{product.type},
				{product.description},
				{product.discount},
				{product.taxes},
				{product.subtotal},
				{product.total},
				{product.created_at}
			</Text>
		</View>
	);
};


const styles = StyleSheet.create({
	container: {

	},
});
