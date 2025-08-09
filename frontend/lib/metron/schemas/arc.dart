class Arc {
  final int id;
  final String name;
  final String image;
  final int cvId;
  final int gcdId;
  final String resourceUrl;
  final String modified;

  Arc({
    required this.id,
    required this.name,
    required this.image,
    required this.cvId,
    required this.gcdId,
    required this.resourceUrl,
    required this.modified,
  });

  factory Arc.fromJson(Map<String, dynamic> json) {
    return Arc(
      id: json['id'],
      name: json['name'],
      image: json['image'],
      cvId: json['cv_id'],
      gcdId: json['gc_id'],
      resourceUrl: json['resource_url'],
      modified: json['modified'],
    );
  }
}
