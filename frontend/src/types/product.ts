export type Product = {
	id: string,
	stock: number,
	brand: string,
	type: string,
	description?: string | null,
	discount: number,
	taxes: number,
	subtotal: number,
	total: number,
	created_at: string,
}

export type UpdateProduct = {
	stock?: number,
	brand?: string,
	type?: string,
	description?: string | null,
	discount?: number,
	taxes?: number,
	subtotal?: number,
	total?: number,
	created_at?: string,
}
