class Publisher {
  final int id;
  final String name;
  final int founded;
  final String country;
  final String desc;
  final String image;
  final int cvId;
  final int gcdId;
  final String resourceUrl;
  final String modified;

  Publisher({
    required this.id,
    required this.name,
    required this.founded,
    required this.country,
    required this.desc,
    required this.image,
    required this.cvId,
    required this.gcdId,
    required this.resourceUrl,
    required this.modified,
  });

  factory Publisher.fromJson(Map<String, dynamic> json) {
    return Publisher(
      id: json['id'],
      name: json['name'],
      founded: json['founded'],
      country: json['country'],
      desc: json['desc'],
      image: json['image'],
      cvId: json['cv_id'],
      gcdId: json['gcd_id'],
      resourceUrl: json['resource_url'],
      modified: json['modified'],
    );
  }
}
