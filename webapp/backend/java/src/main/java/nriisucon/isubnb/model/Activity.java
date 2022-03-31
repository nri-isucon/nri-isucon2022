package nriisucon.isubnb.model;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@NoArgsConstructor
@AllArgsConstructor
public class Activity {
    private String id;
    private String name;
    private String address;
    private String location;
    private int maxPeopleNum;
    private String description;
    private String catchPhrase;
    private String attribute;
    private String category;
    private int price;
    private String photo_1;
    private String photo_2;
    private String photo_3;
    private String photo_4;
    private String photo_5;
    private double rate;
    private String ownerId;
}
