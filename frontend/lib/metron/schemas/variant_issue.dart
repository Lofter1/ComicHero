class VariantIssue {
  final String name;
  final String sku;
  final String upc;
  final String image;

  VariantIssue({
    required this.name,
    required this.sku,
    required this.upc,
    required this.image,
  });

  factory VariantIssue.fromJson(Map<String, dynamic> json) {
    return VariantIssue(
      name: json['name'],
      sku: json['sku'],
      upc: json['upc'],
      image: json['image'],
    );
  }
}
